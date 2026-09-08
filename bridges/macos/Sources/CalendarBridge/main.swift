import EventKit
import EventKitCore
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
    let knownSourceHash: String?
    let forceUpdate: Bool?
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
    var syncedIds: [String] = []
}

let input = FileHandle.standardInput.readDataToEndOfFile()
let request = try decodeBridgeRequest(SyncRequest.self, from: input)

let store = EKEventStore()
let granted = try requestPermissions([.event], from: store)[0]
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

func academicYear(from id: String) -> String? {
    guard id.hasPrefix("academic:") else { return nil }
    let parts = id.split(separator: ":", omittingEmptySubsequences: false)
    guard parts.count > 1 else { return nil }
    return String(parts[1]).trimmingCharacters(in: .whitespacesAndNewlines)
}

func fallbackKey(for id: String, title: String) -> String? {
    guard let year = academicYear(from: id) else { return nil }
    let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmedTitle.isEmpty else { return nil }
    return "academic-fallback:\(year):\(trimmedTitle)"
}

func calendarSearchRange(for events: [AcademicEvent]) -> (Date, Date) {
    let calendar = Calendar.current
    let startDates = events.map { $0.startAt }
    guard let minDate = startDates.min(), let maxDate = startDates.max() else {
        let now = Date()
        return (now, calendar.date(byAdding: .year, value: 1, to: now) ?? now)
    }
    let minYear = calendar.component(.year, from: minDate)
    let maxYear = calendar.component(.year, from: maxDate)
    let start = calendar.date(from: DateComponents(year: minYear, month: 1, day: 1)) ?? minDate
    let end = calendar.date(from: DateComponents(year: maxYear + 1, month: 12, day: 31)) ?? (calendar.date(byAdding: .year, value: 1, to: maxDate) ?? maxDate)
    return (start, end)
}

let calendarName = request.calendarName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Kwangwoon Univ." : request.calendarName
let calendar = try klapCalendar(named: calendarName, useExistingList: request.useExistingList)

let (rangeStart, rangeEnd) = calendarSearchRange(for: request.events)
let predicate = store.predicateForEvents(withStart: rangeStart, end: rangeEnd, calendars: store.calendars(for: .event))
var known: [String: EKEvent] = [:]
var fallbackKnown: [String: EKEvent] = [:]
for event in store.events(matching: predicate) {
    guard let id = klapID(from: event), known[id] == nil else { continue }
    known[id] = event
    if let key = fallbackKey(for: id, title: event.title), fallbackKnown[key] == nil {
        fallbackKnown[key] = event
    }
}

var result = SyncResult()
for item in request.events {
    let fallback = fallbackKey(for: item.id, title: item.title)
    let existing = known[item.id] ?? (fallback.flatMap { fallbackKnown[$0] })
    let hasKnownSource = item.knownSourceHash?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
    if existing != nil && !hasKnownSource && item.forceUpdate != true {
        result.skipped += 1
        continue
    }
    if existing == nil && hasKnownSource && item.forceUpdate != true {
        result.skipped += 1
        continue
    }
    let event = existing ?? EKEvent(eventStore: store)
    let isNew = existing == nil
    event.calendar = calendar
    if isNew || item.forceUpdate == true {
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
    }
    let saveSpan: EKSpan = (event.recurrenceRules?.isEmpty == false) ? .futureEvents : .thisEvent
    try store.save(event, span: saveSpan, commit: false)
    if isNew {
        result.created += 1
    } else if item.forceUpdate == true {
        result.updated += 1
    } else {
        result.skipped += 1
    }
    result.syncedIds.append(item.id)
}

try store.commit()

let data = try encodeBridgeResponse(result)
FileHandle.standardOutput.write(data)
