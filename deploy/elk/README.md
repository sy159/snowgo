## elk单节点部署安装
这里使用单节点es8.14.1部署，基于filebeat+es+kibana完成日志收集存储到kibana展示统计
### 1. 根据需求修改kibana配置
```shell
vim elasticsearch/elasticsearch.yml  # es配置
vim kibana/kibana.yml  # kibana配置
vim filebeat/filebeat.yml  # filebeat配置
```

### 2 设置配置文件权限
```shell
chmod 777 -R elasticsearch  # 给es授权
chmod 777 -R kibana  # 给kibana授权
chmod go-w filebeat/filebeat.yml  # 给filebeat授权
```

### 3 启动服务
```shell
docker-compose up -d
```

### 4 给kibana设置链接es的用户以及密码
- 通过es的api修改密码(推荐)
  - 使用系统默认默认用户(推荐，默认kibana_system)
    ```shell
    docker-compose exec elasticsearch bash
    curl -u "elastic:你配置的ELASTIC_PASSWORD" \
     -X POST "http://127.0.0.1:9200/_security/user/kibana_system/_password" \
     -H "Content-Type: application/json" \
     -d '{"password": "zx.123"}'
    ```
  - 使用不存在的用户
    ```shell
    docker-compose exec elasticsearch bash
    curl -u "elastic:你配置的ELASTIC_PASSWORD" \
     -X POST "http://127.0.0.1:9200/_security/user/你配置的KIBANA_USER" \
     -H "Content-Type: application/json" \
     -d '{"password": "你配置的KIBANA_PASSWORD", "roles": ["kibana_system"]}'
    ```
- 进入容器执行命令修改密码
  ```shell
  docker-compose exec elasticsearch bash  # 进入es容器，根据情况调整
  cd bin/
  elasticsearch-setup-passwords interactive  # 修改密码
  ```
