## Docker

docker attach <container_id>

Or use an interactive shell:

docker exec -it <container_id> /bin/sh  # Use /bin/bash if available


docker logs --since 5m <container_id>

```bash
docker stop $(docker ps -aq) // stop containers
docker rm $(docker ps -aq) // removes containers
docker rmi $(docker images -aq) // removes images
```

#### GOAT:

docker-compose -f docker-compose.dev.yml up --build



## Transfer Files to Server:

scp -i pvp.pem -r C:\Projetos\SSH-AWS\damas-server\SSL\* ec2-user@remote_host:/home/ec2-user/ssl/

