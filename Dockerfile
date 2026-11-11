FROM node:24-alpine AS frontend-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27.4-alpine AS backend-build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/flashcart ./cmd/flashcart

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=backend-build /out/flashcart /app/flashcart
COPY --from=backend-build /src/backend/migrations /app/migrations
COPY --from=frontend-build /src/frontend/dist /app/static
ENV APP_ENV=production PORT=8080 FRONTEND_DIST=/app/static
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/flashcart"]
CMD ["serve"]
