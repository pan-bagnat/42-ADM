# syntax=docker/dockerfile:1.7

FROM node:20-alpine AS admin-deps
WORKDIR /src/admin
COPY frontend-admin/package.json frontend-admin/package-lock.json* frontend-admin/pnpm-lock.yaml* frontend-admin/yarn.lock* ./
RUN \
  if [ -f package-lock.json ]; then npm ci; \
  elif [ -f pnpm-lock.yaml ]; then npm install -g pnpm && pnpm install --frozen-lockfile; \
  elif [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
  else npm install; fi
COPY frontend-admin/ ./
ARG ADMIN_BACKEND_URL=/api
ARG ADMIN_BASE_PATH=/admin/
ENV VITE_BACKEND_URL=${ADMIN_BACKEND_URL}
ENV VITE_BASE_PATH=${ADMIN_BASE_PATH}
RUN npm run build

FROM node:20-alpine AS student-deps
WORKDIR /src/student
COPY frontend-student/package.json frontend-student/package-lock.json* frontend-student/pnpm-lock.yaml* frontend-student/yarn.lock* ./
RUN \
  if [ -f package-lock.json ]; then npm ci; \
  elif [ -f pnpm-lock.yaml ]; then npm install -g pnpm && pnpm install --frozen-lockfile; \
  elif [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
  else npm install; fi
COPY frontend-student/ ./
ARG STUDENT_BACKEND_URL=/api
ARG STUDENT_BASE_PATH=/
ENV VITE_BACKEND_URL=${STUDENT_BACKEND_URL}
ENV VITE_BASE_PATH=${STUDENT_BASE_PATH}
RUN npm run build

FROM nginx:1.25-alpine AS runner
COPY nginx/nginx.conf /etc/nginx/nginx.conf
COPY --from=student-deps /src/student/dist/ /usr/share/nginx/html/
COPY --from=admin-deps /src/admin/dist/ /usr/share/nginx/html/admin/
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
