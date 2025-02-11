FROM node:alpine

WORKDIR /app

CMD npm install && npm run build