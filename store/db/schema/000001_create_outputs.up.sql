CREATE TABLE IF NOT EXISTS `outputs` (
    `sequence` bigint NOT NULL,
    `created_at` datetime NOT NULL,
    `hash` varchar(64) NOT NULL,
    `index` tinyint NOT NULL,
    `asset_id` char(36) NOT NULL,
    `amount` decimal(64,8) NOT NULL,
    PRIMARY KEY (`sequence`),
    INDEX `idx_outputs_asset` (`asset_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;