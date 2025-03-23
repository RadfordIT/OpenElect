FROM node:latest
WORKDIR /OpenElect
RUN npm install -g tailwindcss@3
COPY . .
COPY ./config/colors.json /tmp/colors.json
COPY ./config/config.yaml /tmp/config.yaml
CMD ["npx", "tailwindcss", "-i", "./css/styles.css", "-o", "./css/output.css", "--watch"]