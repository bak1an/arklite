CREATE TABLE `dev_table` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `bigint_column` bigint unsigned NOT NULL,
  `timestamp_start_column` int NOT NULL,
  `timestamp_end_column` int NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;