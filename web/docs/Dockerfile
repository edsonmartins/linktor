# Documentação (Docusaurus): build estático servido por nginx.
# 24 (npm 11) é o ambiente em que o package-lock.json é gerado — o mesmo do CI.
FROM node:24-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .

# Endereço público do site: entra em canonical, sitemap e nas meta tags, então
# é fixado no build. O padrão fica em docusaurus.config.ts.
ARG DOCS_URL
ENV DOCS_URL=${DOCS_URL}

RUN npm run build

# Variante sem privilégios do nginx: master e workers rodam como uid 101
# (nginx) e escutam em 8080 (não-root não faz bind abaixo de 1024).
FROM nginxinc/nginx-unprivileged:1.27-alpine AS runner

USER root
RUN rm -rf /usr/share/nginx/html/*
COPY --from=builder /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
USER nginx

EXPOSE 8080

# 127.0.0.1 e não localhost: no alpine localhost resolve para ::1 primeiro, e o
# nginx só escuta em IPv4 — o healthcheck falharia com o site no ar.
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/ >/dev/null || exit 1
