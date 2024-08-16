CREATE TABLE IF NOT EXISTS `wallets` (
    `id` bigint NOT NULL AUTO_INCREMENT,
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `user_id` char(36) NOT NULL,
    `label` varchar(64) NOT NULL,
    `session_id` char(36) NOT NULL,
    `pin_token` varchar(64) NOT NULL,
    `pin` varchar(64) NOT NULL,
    `private_key` varchar(128) NOT NULL,
    `spend_key` char(64) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY idx_wallets_user (`user_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
