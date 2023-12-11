package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
	"github.com/pandodao/safe-wallet/handler/rpc/safewallet"
	"github.com/pandodao/safe-wallet/store"
	"github.com/shopspring/decimal"
	"github.com/twitchtv/twirp"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Config struct {
	ClientID string `valid:"uuid,required"`
}

func New(
	outputs core.OutputStore,
	transfers core.TransferStore,
	transferz core.TransferService,
	logger *slog.Logger,
	cfg Config,
) *Server {
	if _, err := govalidator.ValidateStruct(cfg); err != nil {
		panic(err)
	}

	return &Server{
		outputs:   outputs,
		transfers: transfers,
		transferz: transferz,
		logger:    logger.With("server", "rpc"),
		sf:        &singleflight.Group{},
		addr:      mixin.RequireNewMixAddress([]string{cfg.ClientID}, 1),
	}
}

type Server struct {
	outputs   core.OutputStore
	transfers core.TransferStore
	transferz core.TransferService
	logger    *slog.Logger
	sf        *singleflight.Group
	addr      *mixin.MixAddress
}

func (s *Server) Handler() (string, http.Handler) {
	svr := safewallet.NewSafeWalletServiceServer(s, nil)
	return svr.PathPrefix(), svr
}

func (s *Server) FindTransfer(ctx context.Context, req *safewallet.FindTransferRequest) (*safewallet.FindTransferResponse, error) {
	transfer, err := s.transfers.FindTrace(ctx, req.TraceId)
	if err != nil {
		if store.IsErrNotFound(err) {
			return nil, twirp.NotFoundError("transfer not found")
		}

		return nil, err
	}

	return &safewallet.FindTransferResponse{Transfer: viewTransfer(transfer)}, nil
}

func (s *Server) CreateTransfer(ctx context.Context, req *safewallet.CreateTransferRequest) (*safewallet.CreateTransferResponse, error) {
	transfer := &core.Transfer{
		TraceID:  req.TraceId,
		Status:   core.TransferStatusPending,
		AssetID:  req.AssetId,
		Amount:   generic.Try(decimal.NewFromString(req.Amount)),
		Memo:     req.Memo,
		Opponent: generic.Try(mixin.NewMixAddress(req.Opponents, uint8(max(req.Threshold, 1)))),
	}

	if _, err := uuid.Parse(req.TraceId); err != nil {
		return nil, twirp.InvalidArgument.Error("invalid trace id")
	}

	if _, err := uuid.Parse(req.AssetId); err != nil {
		return nil, twirp.InvalidArgument.Error("invalid asset id")
	}

	if !transfer.Amount.IsPositive() || transfer.Amount.Truncate(8).LessThan(transfer.Amount) {
		return nil, twirp.InvalidArgument.Error("invalid amount")
	}

	if len(transfer.Memo) > 200 {
		return nil, twirp.InvalidArgument.Error("memo too long")
	}

	if transfer.Opponent == nil {
		return nil, twirp.InvalidArgument.Error("invalid opponents & threshold")
	}

	v, err, _ := s.sf.Do(transfer.TraceID, func() (interface{}, error) {
		return transfer, s.createTransfer(ctx, transfer)
	})

	if err != nil {
		return nil, err
	}

	return &safewallet.CreateTransferResponse{Transfer: viewTransfer(v.(*core.Transfer))}, nil
}

func (s *Server) inspectTransferStatus(ctx context.Context, traceID string) (core.TransferStatus, error) {
	transfer, err := s.transferz.Find(ctx, traceID)

	switch {
	case err == nil && transfer.Status > core.TransferStatusPending:
		return transfer.Status, nil
	case err != nil && !mixin.IsErrorCodes(err, mixin.EndpointNotFound):
		return 0, err
	default:
		return core.TransferStatusPending, nil
	}
}

func (s *Server) createTransfer(ctx context.Context, transfer *core.Transfer) error {
	logger := s.logger.With("transfer", transfer.TraceID)

	if status, err := s.inspectTransferStatus(ctx, transfer.TraceID); err != nil {
		logger.Error("inspectTransferStatus", "err", err)
		return err
	} else if status > core.TransferStatusPending {
		logger.Debug("transfer already handled", "status", status)
		transfer.Status = status
		return nil
	}

	offset, err := s.transfers.GetAssignOffset(ctx, transfer.AssetID)
	if err != nil {
		logger.Error("transfers.GetAssignOffset", "err", err)
	}

	const limit = 256
	outputs, err := s.outputs.List(ctx, offset, transfer.AssetID, transfer.Amount, limit)
	if err != nil {
		logger.Error("outputs.List", "err", err)
		return err
	}

	if len(outputs) == 0 {
		return twirp.Aborted.Error("insufficient pool").
			WithMeta("code", strconv.Itoa(mixin.InsufficientBalance))
	}

	var (
		sum    decimal.Decimal
		ranges [2]uint64
	)

	ranges[0] = outputs[0].Sequence
	for _, output := range outputs {
		sum = sum.Add(output.Amount)
		ranges[1] = output.Sequence
	}

	if sum.LessThan(transfer.Amount) {
		logger.Debug("insufficient balance", "got", sum, "want", transfer.Amount)

		if len(outputs) == limit {
			memo := fmt.Sprintf("merge from %d to %d", ranges[0], ranges[1])
			trace := uuid.NewSHA1(uuid.NameSpaceOID, []byte(memo))

			logger = logger.With("merge", trace.String())
			logger.Debug("limit reached ,try merge outputs")

			merge := &core.Transfer{
				TraceID:     trace.String(),
				Status:      core.TransferStatusPending,
				AssetID:     transfer.AssetID,
				Amount:      sum,
				Memo:        memo,
				Opponent:    s.addr,
				AssignRange: ranges,
			}

			if err := s.transfers.Assign(ctx, merge, offset); err != nil {
				logger.Error("transfers.Assign", "err", err)
			}
		}

		return twirp.Aborted.Error("insufficient balance").
			WithMeta("code", strconv.Itoa(mixin.InsufficientBalance))
	}

	transfer.AssignRange = ranges
	if err := s.transfers.Assign(ctx, transfer, offset); err != nil {
		logger.Error("transfers.Assign", "err", err)
		return err
	}

	return nil
}

func viewTransfer(transfer *core.Transfer) *safewallet.Transfer {
	return &safewallet.Transfer{
		TraceId:   transfer.TraceID,
		CreatedAt: timestamppb.New(transfer.CreatedAt),
		Status:    safewallet.Transfer_Status(transfer.Status),
		AssetId:   transfer.AssetID,
		Amount:    transfer.Amount.String(),
		Memo:      transfer.Memo,
		Opponents: transfer.Opponent.Members(),
		Threshold: uint32(transfer.Opponent.Threshold),
	}
}
