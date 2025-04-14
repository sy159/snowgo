# 创建user表
CREATE TABLE `user` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(20) NOT NULL,
  `password` varchar(64) NOT NULL,
  `tel` varchar(20) NOT NULL,
  `sex` char(1) DEFAULT 'M' COMMENT 'M表示男，F表示女',
  `wallet_amount` decimal(12,2) DEFAULT '0.00',
  `is_delete` tinyint(1) DEFAULT '0' COMMENT '1表示用户已经被删除，0表示可用',
  `created_at` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` datetime(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  INDEX idx_username (username),
  UNIQUE INDEX uniq_tel (tel)
) ENGINE=InnoDB AUTO_INCREMENT=47 DEFAULT CHARSET=utf8mb4;

# 插入测试数据
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (1, 'Heung Ming', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000001', 'M', 971.13, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (2, 'Tang Ziyi', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000002', 'M', 453.45, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (3, 'Chu Chi Yuen', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000003', 'M', 120.28, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (4, 'Nicholas Mitsuki', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000004', 'F', 952.98, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (5, 'Nicholas Phillips', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000005', 'M', 361.66, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (6, 'Li Lu', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000006', 'F', 887.29, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (7, 'Susan Rogers', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000007', 'F', 634.32, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (8, 'Chan Wai Yee', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000008', 'F', 476.60, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (9, 'Loui Wing Suen', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18700000009', 'F', 961.54, 0);
INSERT INTO `user` (`id`, `username`, `password`, `tel`, `sex`, `wallet_amount`, `is_delete`) VALUES (10, 'Sato Nanami', '504d39d5009c0ab8127e0ca48e43dd5b4f66ff823e80d5d5c89f9e64a4792f73', '18712345678', 'M', 519.69, 0);
