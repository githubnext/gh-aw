# Feature Example: Task Management System

## Feature Description

Create a simple task management system that allows users to:
- Create tasks with title, description, and priority
- Mark tasks as complete or incomplete
- Filter tasks by status (all, active, completed)
- Delete tasks

## Requirements

1. **Task Creation**
   - Users should be able to create new tasks
   - Each task must have a title (required)
   - Each task can have an optional description
   - Priority levels: low, medium, high

2. **Task Status Management**
   - Tasks can be marked as complete or incomplete
   - Visual indication of task status
   - Ability to toggle status

3. **Task Filtering**
   - View all tasks
   - View only active (incomplete) tasks
   - View only completed tasks

4. **Task Deletion**
   - Users can delete tasks
   - Confirmation before deletion

## Technical Context

- Frontend: React with TypeScript
- State Management: React Context API
- Storage: LocalStorage for persistence
- Styling: Tailwind CSS

## Expected Outcome

A functional task management application with a clean, intuitive interface that persists data across browser sessions.
