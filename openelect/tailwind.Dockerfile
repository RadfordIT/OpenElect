FROM node:latest
WORKDIR /app

RUN npm install -g tailwindcss
COPY . .

CMD ["npx", "tailwindcss", "-i", "./css/styles.css", "-o", "./css/output.css", "--watch"]