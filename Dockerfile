ARG PLAYWRIGHT_VERSION=1.61.0-noble
FROM mcr.microsoft.com/playwright:v${PLAYWRIGHT_VERSION}

LABEL org.label-schema.description="Playwright screenshot image" \
      org.label-schema.name="ubuntu" \
      org.label-schema.schema-version="" \
      org.label-schema.usage="" \
      org.label-schema.vcs-url="" \
      org.label-schema.vendor="" \
      org.label-schema.version="latest"

RUN apt-get update && apt-get install -y netcat-openbsd

WORKDIR /app
COPY package.json .
RUN npm install

COPY . public/ server/ capture/ .
RUN chmod +x start.sh

CMD [ "/app/start.sh" ]
