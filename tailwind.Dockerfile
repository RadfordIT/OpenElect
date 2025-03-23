FROM node:latest
WORKDIR /app

RUN npm install -g tailwindcss@3
COPY . .
WORKDIR /
COPY ./config/colors.json /tmp/colors.json
COPY ./config/config.yaml /tmp/config.yaml
WORKDIR /app
CMD ["npx", "tailwindcss", "-i", "./css/styles.css", "-o", "./css/output.css", "--watch"]