# Learning Tools Interoperability v1.3 Showcase (Go)

Simplified monorepo for an LTI 1.3 Platform (Go), a sample Tool (ltijs/Node), and a small React FE to drive launches.

This repository is intended for audience who seeks for complete implementation of LTI as platform, and end-to-end PoC to integrate with other LTI Tools (like H5P, Moodle, etc).  You can use this project as sandbox to play integration with other Tool,  or you can directly clone the go backend folder, and start customizing it as part of your LMS services, to start integrating with LTI Tools.

**Implemented LTI specifications for LMS:**
- OIDC
- OAuth2
- Deep Linking
- Resource Link Launch,
- AGS
- NRPS

## Setup
### NGINX
The final result will be a website, i.e. **https://monarch-legal-admittedly.ngrok-free.app/**  that can be opened from your web browser, and it will showcase integration with LTI tools. In order to do this, we need to install locally NGINX that hosts:
* Go backend
* Vite React frontend
* Ltijs (nodeJS library implementation that can simulate Tool integration that is difficult to find on real, free to use LTI tools, like testing AGS and NRPS endpoints)
After you have NGINX properly installed, use the file **config/vite-lti.conf** and put it into your  /etc/nginx/sites-enabled/ 
If you have existing file on sites-enabled please consider move them first to /etc/nginx/sites-available/

With this NGINX settings, we will have a **https://monarch-legal-admittedly.ngrok-free.app/** where:
- port 80 and 443 (SSL) are served by ngrok service
- path /  (root) is served by our Vite FE
- path /api  is served by our Go backend
- path /tool is served by our ltijs application

### NGROK
Sign up for free account at  [https://dashboard.ngrok.com/signup](https://dashboard.ngrok.com/signup)
Then you will be given a free sub domain that can be accessed from direct internet later on.
Follow the documentation provided there to install ngrok app locally in your PC.
Finally run the ngrok:
```
    ngrok config add-authtoken <your ngrok token here>;
    ngrok http --url=<your ngrok free sub-domain here> 80
```
Note:  on **config/vite-lti.conf** you need to update the host according to your ngrok sub-domain.
And then you can restart NGINX:
```
sudo service nginx restart
```

### Go Back End
Go to **be/** folder 
```
export PLATFORM_PRIVATE_KEY_B64='== paste your certificate here=='
export PLATFORM_KID='== paste your key id here =='
export PUBLIC_BASE_URL="https://<your ngrok free sub-domain>"

go mod tidy
go run cmd/server/server.go
```
In order to generate the private key and kid:
```
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out platform_rsa.pem
base64 -w0 platform_rsa.pem > platform_rsa.pem.b64
cat platform_rsa.pem.b64

# for kid you can just generate randomly using uuidgen
uuidgen
```
### Vite Front End
Go to the **fe** folder and do
```
npm install
npm run dev
```


### ltijs application
This is ltijs node library that we utilize to simulate LTI Tool. It helps for us to test LTI endpoints that are difficult to test right away on free websites.
Go to **ltijs** folder:
```
cp .env-example .env
# edit your .env to at least update your ngrok domain
npm install ltijs express dotenv
# ltijs requires mongodb connection (you can customize this if already have existing mongo server)
docker run -d --name ltijs-mongo \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=ltijs \
  -e MONGO_INITDB_ROOT_PASSWORD=ltijsPass \
  mongo:6
```


## Walkthrough
WIP -- Please visit the medium story page soon.

## Repo structure
- `be/` — Go backend (Platform endpoints: OIDC, Deep Linking, NRPS, AGS)
- `fe/` — Vite + React + TypeScript frontend
- `ltijs/` — Sample LTI Tool using ltijs (Node + Express)
- `docs/deep-wiki/` — Project documentation (backend, frontend, ltijs)

## Start here (docs)
- Open: [docs/deep-wiki/Index.md](./docs/deep-wiki/Index.md)
  - Backend: OIDC, Deep Linking, NRPS, AGS
  - Frontend: overview, ToolLaunch page, UI flows
  - ltijs: overview, AGS, NRPS, env & run

