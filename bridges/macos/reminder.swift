import EventKit
import Foundation

struct Assignment: Codable {
    let id: String
    let title: String
    let course: String
    let dueAt: Date?
    let submitted: Bool
    let detailUrl: String
    let notes: String
}

struct SyncRequest: Codable {
    let listName: String
    let useExistingList: Bool
    let alarmBeforeMin: Int
    let assignments: [Assignment]
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
let request = try decoder.decode(SyncRequest.self, from: input)

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

func klapCalendar(named name: String, useExistingList: Bool) throws -> EKCalendar {
    if let existing = store.calendars(for: .reminder).first(where: { $0.title == name }) {
        return existing
    }

    if useExistingList {
        throw NSError(domain: "KLAPReminderBridge", code: 2, userInfo: [
            NSLocalizedDescriptionKey: "Reminder list does not exist: \(name)"
        ])
    }

    let calendar = EKCalendar(for: .reminder, eventStore: store)
    calendar.title = name
    calendar.source = store.sources.first(where: { $0.sourceType == .local }) ?? store.sources.first
    try store.saveCalendar(calendar, commit: true)
    return calendar
}

func existingReminders() -> [EKReminder] {
    let calendars = store.calendars(for: .reminder)
    let predicate = store.predicateForReminders(in: calendars)
    let semaphore = DispatchSemaphore(value: 0)
    var reminders: [EKReminder] = []
    store.fetchReminders(matching: predicate) { found in
        reminders = found ?? []
        semaphore.signal()
    }
    semaphore.wait()
    return reminders
}

func notes(for assignment: Assignment) -> String {
    return assignment.notes
}

func assignmentID(from reminder: EKReminder) -> String? {
    guard let notes = reminder.notes else { return nil }

    if notes.contains("[This reminder is created by KLAP.]") {
        for line in notes.components(separatedBy: .newlines) {
            if line.hasPrefix("ID: ") {
                return String(line.dropFirst(4)).trimmingCharacters(in: .whitespacesAndNewlines)
            }
        }
    }

    guard let rangeStart = notes.range(of: "[KLAP:") else { return nil }
    guard let rangeEnd = notes[rangeStart.upperBound...].range(of: "]") else { return nil }
    return String(notes[rangeStart.upperBound..<rangeEnd.lowerBound])
}

func applyDueDate(_ dueAt: Date, to reminder: EKReminder) {
    let components = Calendar.current.dateComponents([.year, .month, .day, .hour, .minute], from: dueAt)
    reminder.dueDateComponents = components
}

func applyAlarm(_ dueAt: Date, beforeMinutes: Int, to reminder: EKReminder) {
    let offset = max(beforeMinutes, 1)
    guard let alarmDate = Calendar.current.date(byAdding: .minute, value: -offset, to: dueAt) else {
        reminder.alarms = nil
        return
    }
    if alarmDate <= Date() {
        reminder.alarms = nil
        return
    }
    reminder.alarms = [EKAlarm(absoluteDate: alarmDate)]
}

let listName = request.listName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Kwangwoon Univ." : request.listName
let calendar = try klapCalendar(named: listName, useExistingList: request.useExistingList)
var known: [String: EKReminder] = [:]
for reminder in existingReminders() {
    guard let id = assignmentID(from: reminder) else { continue }
    if known[id] == nil {
        known[id] = reminder
    }
}

var result = SyncResult()
for assignment in request.assignments {
    guard let dueAt = assignment.dueAt else {
        result.skipped += 1
        continue
    }

    if let reminder = known[assignment.id] {
        reminder.title = assignment.title
        reminder.notes = notes(for: assignment)
        reminder.url = URL(string: assignment.detailUrl)
        applyDueDate(dueAt, to: reminder)
        applyAlarm(dueAt, beforeMinutes: request.alarmBeforeMin, to: reminder)
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
    reminder.title = assignment.title
    reminder.notes = notes(for: assignment)
    reminder.url = URL(string: assignment.detailUrl)
    applyDueDate(dueAt, to: reminder)
    applyAlarm(dueAt, beforeMinutes: request.alarmBeforeMin, to: reminder)
    try store.save(reminder, commit: false)
    result.created += 1
}

try store.commit()

let encoder = JSONEncoder()
let data = try encoder.encode(result)
FileHandle.standardOutput.write(data)
