FROM rabbitmq:4.1.6-management-alpine

# 将自定义插件复制到容器插件目录中
COPY plugins/*.ez /plugins/

# 复制配置文件（如果有需要）
COPY config/rabbitmq.conf /etc/rabbitmq/
COPY config/definitions.json /etc/rabbitmq/
COPY config/enabled_plugins /etc/rabbitmq/
