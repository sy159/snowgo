SET NAMES utf8mb4;
# 创建user表
DROP TABLE IF EXISTS `sys_user`;
CREATE TABLE `sys_user`
(
    `id`         INT(11)      NOT NULL AUTO_INCREMENT,
    `username`   VARCHAR(64)  NOT NULL COMMENT '登录名，业务唯一',
    `tel`        VARCHAR(20)  NOT NULL COMMENT '手机号码',
    `nickname`   VARCHAR(60)           DEFAULT NULL COMMENT '用户昵称',
    `password`   VARCHAR(255) NOT NULL COMMENT 'pwd',
    `email`      VARCHAR(100)          DEFAULT NULL COMMENT '邮箱',
    `remark`     VARCHAR(500)          DEFAULT NULL COMMENT '备注',
    `status`     TINYINT      NOT NULL DEFAULT 1 COMMENT '用户状态：1 活跃, 2 禁用登录',
    `created_by` INT(11)               DEFAULT NULL COMMENT '创建人 ID',
    `updated_by` INT(11)               DEFAULT NULL COMMENT '更新人 ID',
    `is_deleted` TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '是否删除：0=未删除，1=已删除',
    `deleted_at` DATETIME(6)           DEFAULT NULL COMMENT '删除时间（NULL=未删除）',
    `created_at` DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at` DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username_deleted` (`username`, `is_deleted`),
    UNIQUE KEY `uk_tel_deleted` (`tel`, `is_deleted`),
    KEY `idx_created_by` (`created_by`),
    KEY `idx_status` (`status`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='用户表';

# 创建角色表
DROP TABLE IF EXISTS `sys_role`;
CREATE TABLE `sys_role`
(
    `id`          INT(11)      NOT NULL AUTO_INCREMENT,
    `code`        VARCHAR(64)  NOT NULL COMMENT '角色代码，如 admin、normal',
    `name`        VARCHAR(128) NULL COMMENT '前端展示用名称',
    `description` TEXT         NULL COMMENT '角色描述',
    `created_at`  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at`  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='角色表';

# 创建菜单权限表
DROP TABLE IF EXISTS `sys_menu`;
CREATE TABLE `sys_menu`
(
    `id`         INT(11)                   NOT NULL AUTO_INCREMENT,
    `parent_id`  INT(11)                   NOT NULL DEFAULT 0 COMMENT '父级菜单，0=根节点',
    `menu_type`  ENUM ('Dir','Menu','Btn') NOT NULL COMMENT '类型：Dir/菜单目录, Menu/页面菜单, Btn/按钮操作',
    `name`       VARCHAR(64)               NOT NULL COMMENT '节点名称（前端显示）',
    `path`       VARCHAR(128)              NULL COMMENT '前端路由路径，仅 Dir/Menu 生效',
    `icon`       VARCHAR(64)               NULL COMMENT '节点图标，仅 Dir/Menu 生效',
    `perms`      VARCHAR(100)              NULL COMMENT '权限标识，如 system:user:add，仅 Btn生效',
    `sort_order` INT                       NOT NULL DEFAULT 0 COMMENT '排序号',
    `created_at` DATETIME(6)               NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at` DATETIME(6)               NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='菜单权限表';

# 用户-角色关联表
DROP TABLE IF EXISTS `sys_user_role`;
CREATE TABLE `sys_user_role`
(
    `id`         BIGINT      NOT NULL AUTO_INCREMENT,
    `user_id`    INT(11)     NOT NULL COMMENT '用户ID',
    `role_id`    INT(11)     NOT NULL COMMENT '角色ID',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_role_id` (`role_id`),
    UNIQUE KEY uk_user_role (user_id, role_id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='用户-角色关联表';

# 创建角色-菜单权限表
DROP TABLE IF EXISTS `sys_role_menu`;
CREATE TABLE `sys_role_menu`
(
    `id`         BIGINT      NOT NULL AUTO_INCREMENT,
    `role_id`    INT(11)     NOT NULL COMMENT '角色ID',
    `menu_id`    INT(11)     NOT NULL COMMENT '菜单或按钮ID',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    KEY `idx_role_id` (`role_id`),
    KEY `idx_menu_id` (`menu_id`),
    UNIQUE KEY uk_role_menu (role_id, menu_id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='角色-菜单权限表';

# 创建操作日志表
DROP TABLE IF EXISTS `sys_operation_log`;
CREATE TABLE `sys_operation_log`
(
    `id`            BIGINT      NOT NULL AUTO_INCREMENT COMMENT '日志主键',
    `operator_id`   INT         NOT NULL COMMENT '操作人 ID',
    `operator_name` VARCHAR(64) NOT NULL COMMENT '操作人用户名',
    `operator_type` VARCHAR(32) NOT NULL DEFAULT 'User' COMMENT '操作来源类型，如 User/System/Job/Api',
    `resource`      VARCHAR(32) NOT NULL COMMENT '操作资源类型，如 user/role/menu',
    `resource_id`   INT         NOT NULL COMMENT '资源ID，如 user 表的主键 ID',
    `action`        VARCHAR(10) NOT NULL DEFAULT 'Create' COMMENT '操作类型：Create/Update/Delete，默认 Create',
    `trace_id`      VARCHAR(64) NULL COMMENT '链路id',
    `before_data`   JSON        NULL COMMENT '修改前数据快照（仅 update/delete 时填）',
    `after_data`    JSON        NULL COMMENT '修改后数据快照（仅 create/update 时填）',
    `description`   TEXT        NULL COMMENT '描述',
    `ip`            VARCHAR(45) NULL COMMENT '客户端 IP',
    `created_at`    DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    KEY `idx_operator_id` (`operator_id`),
    KEY `idx_trace_id` (`trace_id`),
    KEY `idx_resource` (`resource`, `resource_id`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='操作日志表';

# 创建系统字典表
DROP TABLE IF EXISTS `sys_dict`;
CREATE TABLE `sys_dict`
(
    `id`          INT(11)      NOT NULL AUTO_INCREMENT,
    `code`        VARCHAR(64)  NOT NULL COMMENT '字典编码',
    `name`        VARCHAR(128) NOT NULL COMMENT '字典名称',
    `description` TEXT         NULL COMMENT '描述',
    `created_at`  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at`  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='系统字典表';

# 创建系统字典枚举值表
DROP TABLE IF EXISTS `sys_dict_item`;
CREATE TABLE `sys_dict_item`
(
    `id`          INT(11)      NOT NULL AUTO_INCREMENT,
    `dict_id`     INT          NOT NULL COMMENT '字典ID',
    `dict_code`   VARCHAR(64)  NOT NULL COMMENT '字典编码',
    `item_name`   VARCHAR(128) NOT NULL COMMENT '枚举显示名称',
    `item_code`   VARCHAR(64)  NOT NULL COMMENT '枚举值编码',
    `status`      VARCHAR(20)  NOT NULL DEFAULT 'Active' COMMENT '状态：Active 启用，Disabled 禁用',
    `sort_order`  INT          NOT NULL DEFAULT 0 COMMENT '排序号',
    `description` TEXT         NULL COMMENT '描述',
    `created_at`  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at`  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_dict_item` (`dict_code`, `item_code`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='系统字典枚举值表';

# 插入测试数据
# 用户数据
INSERT INTO `sys_user` (`username`, `tel`, `nickname`, `password`, `remark`)
VALUES ('admin', '18712345678', '如何好听', '$2a$10$XqU5GKb6wbGXjckKxQtMF.b8nn6MlC17tk2Y.ap//n8swLOQ4fZwO', '管理员'),
       ('test', '18700000001', '只读测试用户', '$2a$10$XqU5GKb6wbGXjckKxQtMF.b8nn6MlC17tk2Y.ap//n8swLOQ4fZwO', '只读测试用户');

# 菜单数据
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (1, 0, 'Dir', '账号管理', '', '', '', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (2, 1, 'Menu', '用户管理', '/account/user', 'fa fa-user-o', '', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (3, 2, 'Btn', '用户列表', '', '', 'account:user:list', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (4, 2, 'Btn', '用户详情', '', '', 'account:user:detail', 2);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (5, 2, 'Btn', '添加用户', '', '', 'account:user:create', 3);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (6, 2, 'Btn', '更新用户', '', '', 'account:user:update', 4);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (7, 2, 'Btn', '删除用户', '', '', 'account:user:delete', 5);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (8, 2, 'Btn', '重置密码', '', '', 'account:user:reset_pwd', 6);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (9, 1, 'Menu', '角色管理', '/account/role', 'fa fa-user-secret', '', 2);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (10, 9, 'Btn', '角色列表', '', '', 'account:role:list', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (11, 9, 'Btn', '角色详情', '', '', 'account:role:detail', 2);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (12, 9, 'Btn', '添加角色', '', '', 'account:role:create', 3);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (13, 9, 'Btn', '更新角色', '', '', 'account:role:update', 4);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (14, 9, 'Btn', '删除角色', '', '', 'account:role:delete', 5);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (15, 1, 'Menu', '菜单管理', '/account/menu', 'fa fa-th-list', '', 3);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (16, 15, 'Btn', '菜单列表', '', '', 'account:menu:list', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (17, 15, 'Btn', '添加菜单', '', '', 'account:menu:create', 2);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (18, 15, 'Btn', '更新菜单', '', '', 'account:menu:update', 3);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (19, 15, 'Btn', '删除菜单', '', '', 'account:menu:delete', 4);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (20, 0, 'Dir', '系统管理', '', '', '', 2);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (21, 20, 'Menu', '操作日志管理', '/system/operation-log', 'fa fa-pencil-square-o', '', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (22, 21, 'Btn', '操作日志列表', '', '', 'system:operation-log:list', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (23, 20, 'Menu', '字典管理', '/system/dict', 'fa fa-bookmark-o', '', 2);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (24, 23, 'Btn', '字典列表', '', '', 'system:dict:list', 1);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (25, 23, 'Btn', '添加字典', '', '', 'system:dict:create', 2);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (26, 23, 'Btn', '更新字典', '', '', 'system:dict:update', 3);
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_type`, `name`, `path`, `icon`, `perms`, `sort_order`)
VALUES (27, 23, 'Btn', '删除字典', '', '', 'system:dict:delete', 3);

# 角色数据
INSERT INTO `sys_role` (`id`, `code`, `name`, `description`)
VALUES (1, 'admin', '管理员', '平台管理员角色'),
       (2, 'read_only', '只读', '仅具备查询与查看权限，不允许任何写操作');

# 角色菜单关联数据
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES (1, 1),
       (1, 2),
       (1, 3),
       (1, 4),
       (1, 5),
       (1, 6),
       (1, 7),
       (1, 8),
       (1, 9),
       (1, 10),
       (1, 11),
       (1, 12),
       (1, 13),
       (1, 14),
       (1, 15),
       (1, 16),
       (1, 17),
       (1, 18),
       (1, 19),
       (1, 20),
       (1, 21),
       (1, 22),
       (1, 23),
       (1, 24),
       (1, 25),
       (1, 26),
       (1, 27),
       # 只读
       (2, 1),
       (2, 2),
       (2, 3),
       (2, 4),
       (2, 9),
       (2, 10),
       (2, 11),
       (2, 15),
       (2, 16),
       (2, 20),
       (2, 21),
       (2, 22),
       (2, 23),
       (2, 24);

# 用户角色关联数据
INSERT INTO `sys_user_role` (`user_id`, `role_id`)
VALUES (1, 1);
INSERT INTO `sys_user_role` (`user_id`, `role_id`)
VALUES (2, 2);

# 系统字典数据
INSERT INTO `sys_dict` (`id`, `code`, `name`, `description`)
VALUES (1, 'status', '状态', '统一状态枚举'),
       (2, 'operation_resource', '资源', '操作的资源枚举（用于操作日志等）');

# 系统字典项数据
INSERT INTO `sys_dict_item` (`dict_id`, `dict_code`, `item_name`, `item_code`, `status`, `sort_order`, `description`)
VALUES (1, 'status', '启用', 'Active', 'Active', 0, '状态-启用'),
       (1, 'status', '禁用', 'Disabled', 'Active', 0, '状态-禁用'),
       (2, 'operation_resource', '用户', 'User', 'Active', 0, '操作资源-用户相关'),
       (2, 'operation_resource', '角色', 'Role', 'Active', 0, '操作资源-角色相关'),
       (2, 'operation_resource', '菜单', 'Menu', 'Active', 0, '操作资源-菜单相关'),
       (2, 'operation_resource', '字典', 'Dict', 'Active', 0, '操作资源-系统字典相关'),
       (2, 'operation_resource', '字典枚举', 'DictItem', 'Active', 0, '操作资源-系统字典枚举相关');
