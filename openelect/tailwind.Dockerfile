FROM node:latest
WORKDIR /app

RUN npm install -g tailwindcss@3
COPY . .

CMD ["npx", "tailwindcss", "-i", "./css/styles.css", "-o", "./css/output.css", "--watch"]