# Database Schema

This document describes the database schema for Monocle, designed for use with GORM and PostgreSQL.

## Entity Relationship Diagram

```mermaid
erDiagram
    USERS {
        uint id PK
        string email "unique, not null"
        string password_hash "not null"
        string name "not null"
        time created_at
        time updated_at
        time deleted_at
    }

    PROJECTS {
        uint id PK
        string name "not null"
        string description
        uint owner_id FK "references users(id)"
        time created_at
        time updated_at
        time deleted_at
    }

    PROJECT_MEMBERSHIPS {
        uint id PK
        uint user_id FK "references users(id)"
        uint project_id FK "references projects(id)"
        string role "admin, member, viewer"
        time created_at
        time updated_at
        time deleted_at
    }

    MONITORS {
        uint id PK
        uint project_id FK "references projects(id)"
        string name "not null"
        string type "http, ping, database, etc"
        jsonb config
        string status "active, paused, error"
        int interval "seconds"
        time created_at
        time updated_at
        time deleted_at
    }

    MONITOR_CHECKS {
        uint id PK
        uint monitor_id FK "references monitors(id)"
        string status "success, failure, timeout"
        int response_time "milliseconds"
        string message
        time checked_at
        time created_at
        time updated_at
        time deleted_at
    }

    INCIDENTS {
        uint id PK
        uint monitor_id FK "references monitors(id)"
        string status "open, investigating, resolved"
        string severity "low, medium, high, critical"
        string title "not null"
        string description
        time started_at
        time resolved_at
        time created_at
        time updated_at
        time deleted_at
    }

    NOTIFICATIONS {
        uint id PK
        uint incident_id FK "references incidents(id)"
        uint user_id FK "references users(id)"
        string channel "email, slack, webhook"
        string status "pending, sent, failed"
        string message
        time sent_at
        time created_at
        time updated_at
        time deleted_at
    }

    NOTIFICATION_RULES {
        uint id PK
        uint project_id FK "references projects(id)"
        uint user_id FK "references users(id)"
        string trigger_type "incident_created, incident_resolved"
        string channel "email, slack, webhook"
        jsonb config
        bool is_active "default: true"
        time created_at
        time updated_at
        time deleted_at
    }

    %% Relationships
    USERS ||--o{ PROJECTS : "owns"
    USERS ||--o{ PROJECT_MEMBERSHIPS : "belongs_to"
    USERS ||--o{ NOTIFICATIONS : "receives"
    USERS ||--o{ NOTIFICATION_RULES : "configures"

    PROJECTS ||--o{ PROJECT_MEMBERSHIPS : "has_members"
    PROJECTS ||--o{ MONITORS : "contains"
    PROJECTS ||--o{ NOTIFICATION_RULES : "has_rules"

    MONITORS ||--o{ MONITOR_CHECKS : "has_checks"
    MONITORS ||--o{ INCIDENTS : "generates"

    INCIDENTS ||--o{ NOTIFICATIONS : "triggers"
```

## Model Descriptions

### Users

Stores user authentication and profile information. Each user can own multiple projects and be a member of others through project memberships.

### Projects

Top-level containers for monitoring resources. Each project has an owner and can have multiple team members. Projects group related monitors and define access boundaries.

### Project Memberships

Many-to-many relationship between users and projects. Defines user roles within a project (admin, member, viewer). Roles implicitly define permissions - admins can manage everything, members can create/edit monitors, viewers can only read.

### Monitors

Individual monitoring endpoints or resources (websites, APIs, databases, etc.). The `config` field stores monitor-specific settings like URLs, timeouts, expected responses. The `type` field determines what kind of monitoring is performed.

### Monitor Checks

Historical record of all monitor executions. Stores the result of each check including response time, status, and error messages. Essential for analytics, dashboards, and debugging monitor issues.

### Incidents

Groups related monitor failures into manageable incidents. Supports escalation workflows with severity levels and status tracking. Incidents can be manually created or auto-generated from monitor failures.

### Notifications

Log of all notifications sent to users about incidents. Tracks delivery status across different channels (email, Slack, webhooks) and provides an audit trail.

### Notification Rules

User-configurable rules for when and how to receive notifications. Allows users to customize their alerting preferences per project, including trigger types and delivery channels.

## Implementation Notes

- All tables use `gorm.Model` for ID, timestamps, and soft deletes.
- Foreign keys and relationships are defined for GORM auto-migration.
- Use `datatypes.JSON` for `jsonb` fields.
- Add `gorm` struct tags for constraints and validation.
- Indexes for foreign keys and frequently queried fields are recommended.
