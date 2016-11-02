CREATE TABLE `mgr8_test` (
  `int_id` int(11) unsigned NOT NULL,
  `nullable_int_id` int(11),
  `var_id` varchar(11) DEFAULT NULL,
  `nullable_var_id` varchar(11) NULL DEFAULT '',
  `nonullable_default_var_id` varchar(11) DEFAULT '"what?"',
  `auto_inc_nonnullable` int(11) unsigned NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`int_id`),
  KEY `auto_inc_nonnullable` (`auto_inc_nonnullable`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
