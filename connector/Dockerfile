FROM mcr.microsoft.com/playwright:v1.58.2-noble

RUN apt-get update \
    && apt-get install -y --no-install-recommends fluxbox x11vnc novnc websockify \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install --omit=dev

COPY src ./src
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

EXPOSE 3101
EXPOSE 6080
CMD ["/app/docker-entrypoint.sh"]
