import EventKit
import Foundation

struct Assignment: Codable {
    let id: String
    let title: String
    let course: String
    let dueAt: Date?
    let submitted: Bool
}

struct SyncResult: Codable {
    var created: Int = 0
    var updated: Int = 0
    var completed: Int = 0
    var skipped: Int = 0
}

let input = FileHandle.standardInput.readDataToEndOfFile()
let decoder = JSONDecoder()
decoder.dateDecodingStrategy = .iso8601
let assignments = try decoder.decode([Assignment].self, from: input)

let store = EKEventStore()
let semaphore = DispatchSemaphore(value: 0)
var granted = false
var permissionError: Error?

if #available(macOS 14.0, *) {
    store.requestFullAccessToReminders { ok, error in
        granted = ok
        permissionError = error
        semaphore.signal()
    }
} else {
    store.requestAccess(to: .reminder) { ok, error in
        granted = ok
        permissionError = error
        semaphore.signal()
    }
}

semaphore.wait()
if let permissionError {
    throw permissionError
}
if !granted {
    throw NSError(domain: "KLAPReminderBridge", code: 1, userInfo: [
        NSLocalizedDescriptionKey: "Reminder access was not granted"
    ])
}

func klapCalendar() throws -> EKCalendar {
    if let existing = store.calendars(for: .reminder).first(where: { $0.title == "KLAP" }) {
        return existing
    }
    if let defaultCalendar = store.defaultCalendarForNewReminders() {
        return defaultCalendar
    }

    let calendar = EKCalendar(for: .reminder, eventStore: store)
    calendar.title = "KLAP"
    calendar.source = store.sources.first(where: { $0.sourceType == .local }) ?? store.sources.first
    try store.saveCalendar(calendar, commit: true)
    return calendar
}

func existingReminders(calendar: EKCalendar) -> [EKReminder] {
    let predicate = store.predicateForIncompleteReminders(withDueDateStarting: nil, ending: nil, calendars: [calendar])
    let semaphore = DispatchSemaphore(value: 0)
    var reminders: [EKReminder] = []
    store.fetchReminders(matching: predicate) { found in
        reminders = found ?? []
        semaphore.signal()
    }
    semaphore.wait()
    return reminders
}

func token(for id: String) -> String {
    return "[KLAP:\(id)]"
}

func applyDueDate(_ dueAt: Date, to reminder: EKReminder) {
    let components = Calendar.current.dateComponents([.year, .month, .day, .hour, .minute], from: dueAt)
    reminder.dueDateComponents = components
}

let calendar = try klapCalendar()
var known = Dictionary(uniqueKeysWithValues: existingReminders(calendar: calendar).compactMap { reminder -> (String, EKReminder)? in
    guard let notes = reminder.notes else { return nil }
    guard let rangeStart = notes.range(of: "[KLAP:") else { return nil }
    guard let rangeEnd = notes[rangeStart.upperBound...].range(of: "]") else { return nil }
    let id = String(notes[rangeStart.upperBound..<rangeEnd.lowerBound])
    return (id, reminder)
})

var result = SyncResult()
for assignment in assignments {
    guard let dueAt = assignment.dueAt else {
        result.skipped += 1
        continue
    }

    let marker = token(for: assignment.id)
    if let reminder = known[assignment.id] {
        reminder.title = "\(assignment.course) - \(assignment.title)"
        reminder.notes = marker
        applyDueDate(dueAt, to: reminder)
        if assignment.submitted && !reminder.isCompleted {
            reminder.isCompleted = true
            result.completed += 1
        } else {
            result.updated += 1
        }
        try store.save(reminder, commit: false)
        continue
    }

    if assignment.submitted {
        result.skipped += 1
        continue
    }

    let reminder = EKReminder(eventStore: store)
    reminder.calendar = calendar
    reminder.title = "\(assignment.course) - \(assignment.title)"
    reminder.notes = marker
    applyDueDate(dueAt, to: reminder)
    try store.save(reminder, commit: false)
    result.created += 1
}

try store.commit()

let encoder = JSONEncoder()
let data = try encoder.encode(result)
FileHandle.standardOutput.write(data)
