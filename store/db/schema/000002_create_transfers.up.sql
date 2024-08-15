CREATE TABLE IF NOT EXISTS `transfers` (
    `id` bigint NOT NULL AUTO_INCREMENT,
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `trace_id` char(36) NOT NULL,
    `status` tinyint NOT NULL DEFAULT 0,
    `asset_id` char(36) NOT NULL,
    `amount` decimal(64, 8) NOT NULL,
    `memo` varchar(255) NULL,
    `opponents` varchar(256) NOT NULL,
    `threshold` tinyint NOT NULL DEFAULT 1,
    `output_from` bigint NOT NULL,
    `output_to` bigint NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_transfers_trace` (`trace_id`),
    INDEX `idx_transfers_status` (`status`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 AUTO_INCREMENT = 1;

CREATE TABLE IF NOT EXISTS `assigns` (
    `asset_id` char(36) NOT NULL,
    `offset` bigint NOT NULL,
    `transfer` char(36) NOT NULL,
    PRIMARY KEY (`asset_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
