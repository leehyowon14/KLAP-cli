import EventKit
import Foundation

struct AcademicEvent: Codable {
    let id: String
    let title: String
    let startAt: Date
    let endAt: Date
    let allDay: Bool
    let notes: String
    let url: String
    let recurrence: String?
    let recurrenceEnd: Date?
}

struct SyncRequest: Codable {
    let calendarName: String
    let useExistingList: Bool
    let events: [AcademicEvent]
}

struct SyncResult: Codable {
    var created: Int = 0
    var updated: Int = 0
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
    store.requestFullAccessToEvents { ok, error in
        granted = ok
        permissionError = error
        semaphore.signal()
    }
} else {
    store.requestAccess(to: .event) { ok, error in
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
    throw NSError(domain: "KLAPCalendarBridge", code: 1, userInfo: [
        NSLocalizedDescriptionKey: "Calendar access was not granted"
    ])
}

func klapCalendar(named name: String, useExistingList: Bool) throws -> EKCalendar {
    if let existing = store.calendars(for: .event).first(where: { $0.title == name }) {
        return existing
    }

    if useExistingList {
        throw NSError(domain: "KLAPCalendarBridge", code: 2, userInfo: [
            NSLocalizedDescriptionKey: "Calendar does not exist: \(name)"
        ])
    }

    let calendar = EKCalendar(for: .event, eventStore: store)
    calendar.title = name
    calendar.source = store.sources.first(where: { $0.sourceType == .local }) ?? store.defaultCalendarForNewEvents?.source ?? store.sources.first
    try store.saveCalendar(calendar, commit: true)
    return calendar
}

func klapID(from event: EKEvent) -> String? {
    guard let notes = event.notes else { return nil }
    guard notes.contains("[This calendar event is created by KLAP.]") else { return nil }
    for line in notes.components(separatedBy: .newlines) {
        if line.hasPrefix("ID: ") {
            return String(line.dropFirst(4)).trimmingCharacters(in: .whitespacesAndNewlines)
        }
    }
    return nil
}

let calendarName = request.calendarName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Kwangwoon Univ." : request.calendarName
let calendar = try klapCalendar(named: calendarName, useExistingList: request.useExistingList)

let startDates = request.events.map { $0.startAt }
let rangeStart = startDates.min() ?? Date()
let rangeEnd = Calendar.current.date(byAdding: .year, value: 1, to: startDates.max() ?? Date()) ?? Date()
let predicate = store.predicateForEvents(withStart: rangeStart, end: rangeEnd, calendars: store.calendars(for: .event))
var known: [String: EKEvent] = [:]
for event in store.events(matching: predicate) {
    guard let id = klapID(from: event), known[id] == nil else { continue }
    known[id] = event
}

var result = SyncResult()
for item in request.events {
    let event = known[item.id] ?? EKEvent(eventStore: store)
    let isNew = known[item.id] == nil
    event.calendar = calendar
    event.title = item.title
    event.startDate = item.startAt
    event.endDate = item.endAt
    event.isAllDay = item.allDay
    event.notes = item.notes
    if !item.url.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
        event.url = URL(string: item.url)
    }
    if item.recurrence == "weekly", let recurrenceEnd = item.recurrenceEnd {
        event.recurrenceRules = [
            EKRecurrenceRule(
                recurrenceWith: .weekly,
                interval: 1,
                end: EKRecurrenceEnd(end: recurrenceEnd)
            )
        ]
    } else {
        event.recurrenceRules = nil
    }
    try store.save(event, span: .thisEvent, commit: false)
    if isNew {
        result.created += 1
    } else {
        result.updated += 1
    }
}

try store.commit()

let encoder = JSONEncoder()
let data = try encoder.encode(result)
FileHandle.standardOutput.write(data)
