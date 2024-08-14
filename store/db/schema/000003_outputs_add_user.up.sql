ALTER TABLE
    `outputs`
ADD
    COLUMN `user_id` char(36) NOT NULL
AFTER
    `index`,
ADD
    COLUMN `app_id` char(36) NOT NULL
AFTER
    `user_id`;

ALTER TABLE
    `outputs` DROP INDEX `idx_outputs_asset`;

ALTER TABLE
    `outputs`
ADD
    INDEX `idx_outputs_user_asset` (`user_id`, `asset_id`),
ADD
    INDEX `idx_outputs_app` (`app_id`);

ALTER TABLE
    `transfers`
ADD
    COLUMN `user_id` char(36) NOT NULL BEFORE `asset_id`;

ALTER TABLE
    `assigns` DROP PRIMARY KEY,
ADD
    COLUMN `user_id` char(36) NOT NULL BEFORE `asset_id`,
ADD
    PRIMARY KEY (`user_id`, `asset_id`);
