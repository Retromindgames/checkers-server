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

To restart docker containers in the server 

docker restart $(docker ps -q)


### Clean Up space

git fetch --prune


You can clean up your local Git repository by deleting all branches except the current one and main with these commands:

    git branch | grep -vE "(main|\*)" | xargs git branch -D


#### GOAT:

docker-compose -f docker-compose.dev.yml up --build



## Transfer Files to Server:

scp -i pvp.pem -r C:\Projetos\SSH-AWS\damas-server\SSL\* ec2-user@remote_host:/home/ec2-user/ssl/



## PostgreSQL

### Clean up DB Volume
When the containers start up, if there is an existing database, the init script will be lost.
These are the steps to remove the docker volume to create the database from scratch.

Stop the containers:

```
docker-compose down
```

Remove the volume:

```
docker volume rm go-websocket-checkers_postgres_data
```

and then we start up the containers again:

```
docker-compose up -d
```


### Enter postgres cmd

docker exec -it postgres psql -U sa -d checkers

### Check table names

SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';