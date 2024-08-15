ALTER TABLE
    `outputs` DROP INDEX `idx_outputs_user_asset`,
ADD
    INDEX `idx_outputs_asset` (`asset_id`),
    DROP COLUMN `user_id`;

ALTER TABLE
    `transfers` DROP COLUMN `user_id`;

ALTER TABLE
    `assigns` DROP PRIMARY KEY,
    DROP COLUMN `user_id`,
ADD
    PRIMARY KEY (`asset_id`);
