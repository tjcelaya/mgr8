USE mgr8_db;

CREATE TABLE `mgr8_test` (
  `int_id` int(11) unsigned NOT NULL,
  `nullable_int_id` int(11) DEFAULT NULL,
  `var_id` varchar(11) NOT NULL DEFAULT '',
  `nullable_var_id` varchar(11) DEFAULT '',
  `nonullable_default_var_id` varchar(11) NOT NULL DEFAULT '"what?"',
  `auto_inc_nonnullable_int_id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `nullable_default_var_id` varchar(11) DEFAULT '"what?"',
  `text_default_blob_id` blob,
  PRIMARY KEY `auto_inc_nonnullable` (`auto_inc_nonnullable_int_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;