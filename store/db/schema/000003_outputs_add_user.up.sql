ALTER TABLE
    `outputs`
ADD
    COLUMN `user_id` char(36) NOT NULL
AFTER
    `index`;

ALTER TABLE
    `outputs` DROP INDEX `idx_outputs_asset`;

ALTER TABLE
    `outputs`
ADD
    INDEX `idx_outputs_user_asset` (`user_id`, `asset_id`);

ALTER TABLE
    `transfers`
ADD
    COLUMN `user_id` char(36) NOT NULL
AFTER
    `status`;

ALTER TABLE
    `assigns` DROP PRIMARY KEY,
ADD
    COLUMN `user_id` char(36) NOT NULL FIRST,
ADD
    PRIMARY KEY (`user_id`, `asset_id`);
