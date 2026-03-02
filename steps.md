1. Define backend architecture and folder structure using DDD (domain, application, infrastructure, interfaces/http).
2. Initialize Go module and base app bootstrap (cmd/server, internal packages, config loader).
3. Implement .env config loader with strict validation (DATABASE_URL, UNOSEND_API_KEY, EMAIL_SEND_TIME, EMAIL_SEND_LIMIT, server port, clerk webhook secret).
4. Set up PostgreSQL connection layer and connection health check.
5. Create database schema/migrations for users table with defaults (is_subscribed=true, role=user, total_email_received=0).
6. Implement repository layer with raw SQL for user CRUD and query helpers (subscribed users, stats counts, role-aware lookups).
7. Implement Clerk webhook endpoint to create/update users from Clerk events.
8. Build auth middleware to verify Clerk token/session and map request user.
9. Implement role-based authorization middleware (admin vs user).
10. Create user APIs:
11. GET /me for normal user profile.
12. GET /admin/users for admin list with user info.
13. Create public metadata API (GET /metadata) for unauthenticated stats (total users, total emails sent).
14. Build email service integration with Unosend client abstraction and retry/error handling.
15. Implement daily scheduler using robfig/cron with EMAIL_SEND_TIME.
16. Implement concurrent email dispatch with worker limit from EMAIL_SEND_LIMIT.
17. On successful send, increment total_email_received per user atomically in DB.
18. Add structured logging and request tracing basics (request id, job logs, send failures).
19. Implement graceful shutdown for HTTP server, DB, and cron workers.
20. Add tests for config validation, repository SQL logic, RBAC middleware, and scheduler/email workflow.
21. Add Dockerfile and optional compose for local run.
22. Write backend README with env setup, migration steps, run commands, and cron behavior.

