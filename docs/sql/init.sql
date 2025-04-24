# 创建user表
CREATE TABLE `user`
(
    `id`         INT(11) NOT NULL AUTO_INCREMENT,
    `username`   VARCHAR(64) NOT NULL COMMENT '登录名，业务唯一',
    `tel`        VARCHAR(20) NOT NULL COMMENT '手机号码',
    `nickname`   VARCHAR(60) NULL COMMENT '用户昵称',
    `password`   CHAR(64)    NOT NULL COMMENT 'pwd',
    `status`     ENUM('Active','Disabled') NOT NULL DEFAULT 'Active'
                 COMMENT '状态：Active 活跃，Disabled 禁用登录',
    `is_deleted` TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '是否删除：0=未删除，1=已删除',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    KEY idx_username (`username`),
    KEY idx_tel (`tel`),
    KEY          idx_status (`status`),
    KEY          idx_is_deleted (`is_deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

# 创建角色表
CREATE TABLE `role`
(
    `id`           INT(11)       NOT NULL AUTO_INCREMENT,
    `code`         VARCHAR(64) NOT NULL COMMENT '角色代码，如 admin、normal',
    `name` VARCHAR(128) NULL COMMENT '前端展示用名称',
    `description`  TEXT NULL COMMENT '角色描述',
    `created_at`   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at`   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

# 创建菜单权限表
CREATE TABLE `menu` (
  `id`           INT(11)       NOT NULL AUTO_INCREMENT,
  `parent_id`    INT(11)      NOT NULL  DEFAULT 0   COMMENT '父级菜单，0=根节点',
  `menu_type`    ENUM('Dir','Menu','Btn') NOT NULL COMMENT '类型：Dir/菜单目录, Menu/页面菜单, Btn/按钮操作',
  `name`         VARCHAR(64) NOT NULL COMMENT '节点名称（前端显示）',
  `path`         VARCHAR(128) NULL   COMMENT '前端路由路径，仅 Dir/Menu 生效',
  `icon`         VARCHAR(64)  NULL   COMMENT '节点图标，仅 Dir/Menu 生效',
  `perms`        VARCHAR(100) NULL   COMMENT '权限标识，如 system:user:add，仅 Btn生效',
  `order_num`    INT          NOT NULL DEFAULT 0 COMMENT '排序号',
  `created_at`   DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at`   DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='菜单权限表';

# 用户-角色关联表
CREATE TABLE `user_role` (
  `id`         BIGINT       NOT NULL AUTO_INCREMENT,
  `user_id`    INT(11) NOT NULL COMMENT '用户ID',
  `role_id`    INT(11) NOT NULL COMMENT '角色ID',
  `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户-角色关联表';

# 创建角色-菜单权限表
CREATE TABLE `role_menu` (
  `id`         BIGINT       NOT NULL AUTO_INCREMENT,
  `role_id`    INT(11) NOT NULL COMMENT '角色ID',
  `menu_id`    INT(11) NOT NULL COMMENT '菜单或按钮ID',
  `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_role_id` (`role_id`),
  KEY `idx_menu_id` (`menu_id`)
)  ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色-菜单权限表';


# 插入测试数据
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Shi Xiaoming', '18743585769', '史晓明', 'Kf370RG71fxIUc65iWGFqcRHOs2KHAoZSa96tONX6ECZEulPx8gR4Zzjgy8a', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Lin Yuning', '13446431795', '林宇宁', 'jEwRzhscClfH7V2HBCGpzzrjewgrHWhyadJA9WYjUaUOTySpIR7EehPKNoN2', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Sun Zitao', '18789318732', '孙子韬', 'BXLgZlxBl8vmd5M3Q77b2XyBNzM0cL6zYmXoK2Hy3Y5k3Jp3QsLbV3iOx3Oy', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Gao Yuning', '18746990809', '高宇宁', 'rbbD5MmtNiVTDogHINdc2ZxDCu6Z3d8AYJpnIzr399bAQaaO7IQFJvco7tEp', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Xie Xiaoming', '18782947844', '谢晓明', 'bzemi5P51EPE7KgQjOobsr1I6NDB6Jl9bYKQDzD6eOEq5rtoeKkvY0RZ7JUc', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Duan Yuning', '13421587433', '段宇宁', 'O1pFZfTbY2VGlYS9TnTvrdbuBliopgWNUvPV49g4tImsTkLfrdo9HuEaycFA', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Qian Xiaoming', '13447114679', '钱晓明', 'Mo46wlfxmDkn0pA77SfO7GCoXElPj1tQlsAjpt6qPU4hy5kKMMBVO83ABZVC', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Kong Lu', '13496749646', '孔璐', 'gx0ocyRdQo47TCjGguQl6EaWP8a7XcAtS5m5LHHi5UZ6gfYIlvM8NqjIG6Up', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ('Wu Zhennan', '18794888670', '武震南', '43PA1LvbzkXitvS7hsd93hvsACKwXlixOJTTrjefLLfi00FRIcWe4QzeEf13', 'Active', 0);
INSERT INTO `user` (`username`, `tel`, `nickname`, `password`, `status`, `is_deleted`) VALUES ( 'Hao Zhiyuan', '18726051605', '郝致远', 'PUNVZbDpolL9Wx0h8hLkK0h9qUQcCmABzNK2sD4zIrGD06QxkZtZebRVucXD', 'Active', 0);

