# English Documentation

[Docs Home](../README.md) | [中文](../zh/README.md)

Use this page as the starting point for the English documentation. The root README stays short; these pages hold the detailed feature guides.

## Recommended Reading Path

1. [Overview](./overview.md) — understand the framework shape, feature set, and package layout.
2. [Getting Started](./getting-started.md) — build the first typed API and learn when to use Router vs Controller.
3. [Project and CRUD Scaffolding](./scaffolding.md) — create projects, apps, migrations, and CRUD code with `gin-ninja-cli`.
4. [Data, Binding, and Responses](./data-and-responses.md) — define request inputs, response schemas, pagination, filtering, and sorting.
5. [Middleware and Security](./middleware-security.md) — add auth, sessions, CSRF, security headers, logging, i18n, and upload limits.
6. [Advanced Features](./advanced-features.md) — add caching, API versioning, SSE, and WebSocket endpoints.
7. [Testing APIs with TestClient](./testing.md) — test routers and APIs without manual `httptest` setup.

## Feature Reference

| Topic | Guide |
| --- | --- |
| Architecture and package layout | [Overview](./overview.md) |
| First API, controllers, lightweight structure | [Getting Started](./getting-started.md) |
| CLI startproject/startapp, migrations, CRUD generation | [Project and CRUD Scaffolding](./scaffolding.md) |
| Settings, bootstrap helpers, lifecycle hooks | [Configuration, Bootstrap, and Lifecycle](./configuration.md) |
| Middleware, authentication, sessions, CSRF, security | [Middleware and Security](./middleware-security.md) |
| ModelSchema, response envelope, binding, filtering, sorting | [Data, Binding, and Responses](./data-and-responses.md) |
| Upload/download and OpenAPI operation options | [File Transfer and OpenAPI Controls](./files-and-openapi.md) |
| Cache, API versions, SSE, WebSocket | [Advanced Features](./advanced-features.md) |
| TestClient and API testing | [Testing APIs with TestClient](./testing.md) |
| Admin package and full application examples | [Admin and Full Example](./admin-and-examples.md) |
