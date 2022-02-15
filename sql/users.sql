CREATE TABLE `users` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `imsi` varchar(64) NOT NULL DEFAULT '' COMMENT 'LTE用户唯一标识',
  `mnc` varchar(32) NOT NULL DEFAULT '01' COMMENT '移动网号',
  `mcc` int(11) NOT NULL DEFAULT '86' COMMENT '国家码',
  `apn` varchar(32) NOT NULL DEFAULT 'hebeiyidong' COMMENT 'APN网络',
  `ip` varchar(32) NOT NULL DEFAULT '' COMMENT '分配IP地址',
  `sip_username` varchar(32) NOT NULL DEFAULT '' COMMENT 'SIP网络用户名',
  `sip_dns` varchar(64) NOT NULL DEFAULT '3gpp.net' COMMENT '网络归属',
  `ctime` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
  `utime` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uqidx_imsi` (`imsi`),
  UNIQUE KEY `uqidx_ip` (`ip`),
  UNIQUE KEY `uqidx_sip_username` (`sip_username`),
  KEY `idx_ctime_utime` (`ctime`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4

INSERT INTO `users` (`id`, `imsi`, `mnc`, `mcc`, `apn`, `ip`, `sip_username`, `sip_dns`, `ctime`, `utime`) VALUES (1, '123456789', '01', 86, 'hebeiyidong', '2.2.2.2', 'jiqimao', '3gpp.net', '2022-01-18 11:48:54', '2022-01-18 11:48:54');