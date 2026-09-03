import EventKit
import Foundation

struct Assignment: Codable {
    let id: String
    let legacyIds: [String]?
    let title: String
    let course: String
    let dueAt: Date?
    let submitted: Bool
    let detailUrl: String
    let notes: String
    let knownSourceHash: String?
    let forceUpdate: Bool?
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
    var syncedIds: [String] = []
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

func replacingAssignmentID(in notes: String?, with id: String) -> String? {
    guard let notes else { return nil }
    var lines = notes.components(separatedBy: .newlines)
    if let index = lines.firstIndex(where: { $0.hasPrefix("ID: ") }) {
        lines[index] = "ID: \(id)"
    }
    return lines.joined(separator: "\n")
}

func legacyResourcePart(from id: String) -> String? {
    guard let separator = id.firstIndex(of: ":") else { return nil }
    let prefix = id[..<separator]
    guard Int(prefix) != nil else { return nil }
    let resourcePart = id[id.index(after: separator)...]
    return resourcePart.isEmpty ? nil : String(resourcePart)
}

func reminderCourse(from reminder: EKReminder) -> String? {
    guard let notes = reminder.notes else { return nil }
    for line in notes.components(separatedBy: .newlines) where line.hasPrefix("과목: ") {
        return String(line.dropFirst(4)).trimmingCharacters(in: .whitespacesAndNewlines)
    }
    return nil
}

func legacyMatch(for assignment: Assignment, in known: [String: EKReminder]) -> (String, EKReminder)? {
    let resourceParts = Set((assignment.legacyIds ?? []).compactMap(legacyResourcePart))
    guard !resourceParts.isEmpty else { return nil }
    let matches = known.compactMap { id, reminder -> (String, EKReminder)? in
        guard let resourcePart = legacyResourcePart(from: id),
              resourceParts.contains(resourcePart),
              reminderCourse(from: reminder) == assignment.course else {
            return nil
        }
        return (id, reminder)
    }
    return matches.count == 1 ? matches[0] : nil
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

    let hasKnownSource = assignment.knownSourceHash?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
    let candidateIds = [assignment.id] + (assignment.legacyIds ?? [])
    let exactMatchedId = candidateIds.first(where: { known[$0] != nil })
    let fallbackMatch = exactMatchedId == nil ? legacyMatch(for: assignment, in: known) : nil
    let matchedId = exactMatchedId ?? fallbackMatch?.0
    let matchedReminder = matchedId.flatMap { known[$0] } ?? fallbackMatch?.1
    if let matchedId, let reminder = matchedReminder {
        if !hasKnownSource && assignment.forceUpdate != true {
            result.skipped += 1
            continue
        }
        reminder.calendar = calendar
        if assignment.forceUpdate == true {
            reminder.title = assignment.title
            reminder.notes = notes(for: assignment)
            reminder.url = URL(string: assignment.detailUrl)
            applyDueDate(dueAt, to: reminder)
            applyAlarm(dueAt, beforeMinutes: request.alarmBeforeMin, to: reminder)
        } else if matchedId != assignment.id {
            reminder.notes = replacingAssignmentID(in: reminder.notes, with: assignment.id)
        }
        if assignment.submitted && !reminder.isCompleted {
            reminder.isCompleted = true
            result.completed += 1
        } else {
            if assignment.forceUpdate == true {
                result.updated += 1
            } else {
                result.skipped += 1
            }
        }
        try store.save(reminder, commit: false)
        result.syncedIds.append(assignment.id)
        continue
    }

    if hasKnownSource && assignment.forceUpdate != true {
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
    if assignment.submitted {
        reminder.isCompleted = true
    }
    try store.save(reminder, commit: false)
    result.created += 1
    result.syncedIds.append(assignment.id)
}

try store.commit()

let encoder = JSONEncoder()
let data = try encoder.encode(result)
FileHandle.standardOutput.write(data)
