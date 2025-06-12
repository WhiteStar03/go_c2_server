# Single stage build - serve with Node.js
FROM node:18-alpine

WORKDIR /app

# Install serve globally to serve the React build
RUN npm install -g serve

# Copy package files and install dependencies
COPY package*.json ./
RUN npm ci

# Copy source and build
COPY . .
RUN npm run build

# Expose port 3000
EXPOSE 3000

# Serve the build folder
CMD ["serve", "-s", "build", "-l", "3000"]
