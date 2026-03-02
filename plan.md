## **Daily Durud Reminder - Project Blueprint**

A non-profit, Islamic hobby project designed to send automated daily reminders for Durud Sharif and track user consistency through streaks.

## **Core Features**
- User authentication and creation via `Clerk` and `DB` both.
- Main heart feature: Send email everyday to the all subscrived users (`SELECT * FROM users WHERE user.is_subscribed = true;`) at 10:00 am clock 
- Update users `total_email_received` ++ when each day email is sucessfully sended. 
- Role based authentication: Admin users can see all users with there informations. Normal users can only see there profiles with there data.
- A Metadata api for un authenticated users to show stats count like: Total Users, Total Email Sended etc..

## **Technology**

- Frontend:
  - Nextjs (Modern frontend)
  - TypeScript (TypeSafety)
  - Clerk (Auth)
  - Tailwind CSS (Styling)

- Backend:
  - Golang (High concurrency with DDD - Domain Driven Design pattern)
  - Unosend [ Email sender ](https://www.unosend.co)
  - Clerk (Sync with frontend via webhook for User authorization and creation)
  - Postgress with Neon console (Database)
  - Raw sql query
  - Cron job `robfig/cron`
  - Docker container (optional)

