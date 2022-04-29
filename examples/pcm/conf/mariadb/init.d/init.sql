DROP TABLE IF EXISTS `subscriber`;
CREATE TABLE `subscriber` (
  `msisdn` VARCHAR(40) NOT NULL,
  `name` VARCHAR(40) NOT NULL,
  `value` VARCHAR(255) DEFAULT NULL,
  UNIQUE KEY `profile_key` (`msisdn`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
