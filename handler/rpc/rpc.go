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
	"github.com/zyedidia/generic/mapset"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Config struct {
	ClientID      string   `valid:"required"`
	Prefix        string   `valid:"required"`
	BlockedAssets []string `valid:"uuid"`
}

func New(
	outputs core.OutputStore,
	transfers core.TransferStore,
	wallets core.WalletStore,
	walletz core.WalletService,
	logger *slog.Logger,
	cfg Config,
) *Server {
	if _, err := govalidator.ValidateStruct(cfg); err != nil {
		panic(err)
	}

	return &Server{
		outputs:       outputs,
		transfers:     transfers,
		wallets:       wallets,
		walletz:       walletz,
		logger:        logger.With("server", "rpc"),
		sf:            &singleflight.Group{},
		prefix:        cfg.Prefix,
		blockedAssets: mapset.Of(cfg.BlockedAssets...),
		defaultUserID: cfg.ClientID,
	}
}

type Server struct {
	outputs       core.OutputStore
	transfers     core.TransferStore
	wallets       core.WalletStore
	walletz       core.WalletService
	logger        *slog.Logger
	sf            *singleflight.Group
	blockedAssets mapset.Set[string]
	prefix        string
	defaultUserID string
}

func (s *Server) Handler() (string, http.Handler) {
	svr := safewallet.NewSafeWalletServiceServer(s, twirp.WithServerPathPrefix(s.prefix))
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
	if s.blockedAssets.Has(req.AssetId) {
		return nil, twirp.Aborted.Error("asset is blocked")
	}

	if req.UserId == "" {
		req.UserId = s.defaultUserID
	}

	transfer := &core.Transfer{
		TraceID:  req.TraceId,
		UserID:   req.UserId,
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

func (s *Server) createTransfer(ctx context.Context, transfer *core.Transfer) error {
	logger := s.logger.With("id", transfer.TraceID, "asset", transfer.AssetID, "amount", transfer.Amount)

	if _, err := s.transfers.FindTrace(ctx, transfer.TraceID); err == nil {
		return nil
	} else if !store.IsErrNotFound(err) {
		logger.Error("transfers.FindTrace", "err", err)
		return err
	}

	// if status, err := s.transferz.InspectStatus(ctx, transfer.TraceID); err != nil {
	// 	logger.Error("inspectTransferStatus", "err", err)
	// 	return err
	// } else if status > core.TransferStatusPending {
	// 	logger.Debug("transfer already handled", "status", status)
	// 	transfer.Status = status
	// 	return nil
	// }

	offset, err := s.transfers.GetAssignOffset(ctx, transfer.UserID, transfer.AssetID)
	if err != nil {
		logger.Error("transfers.GetAssignOffset", "err", err)
		return err
	}

	logger.Debug("GetAssignOffset", "offset", offset)

	const limit = 256
	outputs, err := s.outputs.ListTarget(ctx, transfer.UserID, transfer.AssetID, offset, transfer.Amount, limit)
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
				UserID:      transfer.UserID,
				AssetID:     transfer.AssetID,
				Amount:      sum,
				Memo:        memo,
				Opponent:    mixin.RequireNewMixAddress([]string{transfer.UserID}, 1),
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

func (s *Server) CreateWallet(ctx context.Context, req *safewallet.CreateWalletRequest) (*safewallet.CreateWalletResponse, error) {
	s.logger.Debug("create new wallet", "label", req.Label)

	wallet, err := s.walletz.Create(ctx, req.Label)
	if err != nil {
		s.logger.Error("walletz.Create", "err", err)
		return nil, err
	}

	if err := s.wallets.Create(ctx, wallet); err != nil {
		s.logger.Error("wallets.Create", "err", err)
		return nil, err
	}

	return &safewallet.CreateWalletResponse{
		UserId: wallet.UserID,
		Label:  wallet.Label,
	}, nil
}

func (s *Server) FindWallet(ctx context.Context, req *safewallet.FindWalletRequest) (*safewallet.FindWalletResponse, error) {
	balances, err := s.outputs.SumBalances(ctx, req.UserId, "")
	if err != nil {
		s.logger.Error("outputs.SumBalances", "err", err)
		return nil, err
	}

	resp := &safewallet.FindWalletResponse{}
	for _, balance := range balances {
		resp.Balances = append(resp.Balances, &safewallet.Balance{
			AssetId: balance.AssetID,
			Amount:  balance.Amount.String(),
		})
	}

	return resp, nil
}

func viewTransfer(transfer *core.Transfer) *safewallet.Transfer {
	return &safewallet.Transfer{
		TraceId:   transfer.TraceID,
		CreatedAt: timestamppb.New(transfer.CreatedAt),
		Status:    safewallet.Transfer_Status(transfer.Status),
		UserId:    transfer.UserID,
		AssetId:   transfer.AssetID,
		Amount:    transfer.Amount.String(),
		Memo:      transfer.Memo,
		Opponents: transfer.Opponent.Members(),
		Threshold: uint32(transfer.Opponent.Threshold),
	}
}
