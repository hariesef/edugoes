# ltijs Env & Run

Keywords: ltijs, .env, PORT, TOOL_URL, ENCRYPTION_KEY, LTI_KEY, MONGO_URL, npm start, node index.js, nginx, reverse proxy

- Directory: `ltijs/`
- Example env (`.env-example`):
  - `PORT=4000`
  - `TOOL_URL=http://localhost:4000`
  - `ENCRYPTION_KEY=replace-with-long-random`
  - `LTI_KEY=replace-with-long-random`
  - `MONGO_URL=mongodb://ltijs:ltijsPass@localhost:27017/ltijs?authSource=admin`

## Install
```bash
cp .env-example .env
npm install ltijs express dotenv
# optional MongoDB
# docker run -d --name ltijs-mongo -p 27017:27017 mongo:6
```

## Run
```bash
node index.js
# or
npm start
```

## Reverse proxy (Nginx)
```
location /tool/ {
  proxy_pass http://localhost:4000/; # strip /tool
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-For $remote_addr;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_http_version 1.1;
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
}
```

Public endpoints:
- https://<ngrok-domain>/tool/login
- https://<ngrok-domain>/tool/launch
- https://<ngrok-domain>/tool/keys
- https://<ngrok-domain>/tool/.well-known/jwks.json
