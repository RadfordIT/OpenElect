FROM node:latest
WORKDIR /OpenElect
RUN npm install tailwindcss @tailwindcss/cli
RUN npm install -D @tailwindcss/typography
RUN npm i -D daisyui@latest
COPY . .
COPY ./config/colors.json /tmp/colors.json
COPY ./config/config.yaml /tmp/config.yaml
COPY ./config/output.css /tmp/output.css
COPY ./config/styles.css /tmp/styles.css
RUN cat ./config/styles.css
RUN npx @tailwindcss/cli -i ./config/styles.css -o ./config/output.css
CMD ["npx", "@tailwindcss/cli", "-i", "./config/styles.css", "-o", "./config/output.css", "--watch"]
