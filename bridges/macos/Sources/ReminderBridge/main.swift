import EventKit
import EventKitCore
import Foundation

struct Assignment: Codable {
    let id: String
    let legacyIds: [String]?
    let termValue: String?
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
let request = try decodeBridgeRequest(SyncRequest.self, from: input)

let store = EKEventStore()
let granted = try requestPermissions([.reminder], from: store)[0]
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

func metadataRange(in lines: [String]) -> Range<Int>? {
    guard let footerIndex = lines.lastIndex(where: {
        $0.trimmingCharacters(in: .whitespacesAndNewlines) == "[This reminder is created by KLAP.]"
    }) else {
        return nil
    }
    guard let markerIndex = lines[..<footerIndex].lastIndex(where: {
        let marker = $0.trimmingCharacters(in: .whitespacesAndNewlines)
        return marker == "--- KLAP ---" || marker == "========================================="
    }) else {
        return nil
    }
    return lines.index(after: markerIndex)..<footerIndex
}

func assignmentID(from reminder: EKReminder) -> String? {
    guard let notes = reminder.notes else { return nil }

    if notes.contains("[This reminder is created by KLAP.]") {
        let lines = notes.components(separatedBy: .newlines)
        guard let range = metadataRange(in: lines) else { return nil }
        for line in lines[range] {
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
    guard let range = metadataRange(in: lines) else { return notes }
    if let index = lines[range].firstIndex(where: { $0.hasPrefix("ID: ") }) {
        lines[index] = "ID: \(id)"
    }
    return lines.joined(separator: "\n")
}

func legacyResourcePart(from id: String) -> String? {
    let parts = id.split(separator: ":", omittingEmptySubsequences: false)
    if parts.count >= 2, Int(parts[0]) != nil {
        return parts.dropFirst().joined(separator: ":")
    }
    if parts.count >= 3, parts[0] == "lecture", Int(parts[1]) != nil {
        return "lecture:" + parts.dropFirst(2).joined(separator: ":")
    }
    return nil
}

func reminderCourse(from reminder: EKReminder) -> String? {
    guard let notes = reminder.notes else { return nil }
    let lines = notes.components(separatedBy: .newlines)
    guard let range = metadataRange(in: lines) else { return nil }
    for line in lines[range] where line.hasPrefix("과목: ") {
        return String(line.dropFirst(4)).trimmingCharacters(in: .whitespacesAndNewlines)
    }
    return nil
}

struct KnownReminder {
    let id: String
    let reminder: EKReminder
}

func termHashtag(for termValue: String?) -> String? {
    guard let termValue else { return nil }
    let normalized = termValue.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !normalized.isEmpty else { return nil }
    let parts = normalized.split(whereSeparator: { $0 == "," || $0 == "-" })
    if parts.count >= 2 {
        if parts[1] == "3" { return "#\(parts[0])-여름학기" }
        if parts[1] == "4" { return "#\(parts[0])-겨울학기" }
    }
    return "#" + normalized.replacingOccurrences(of: ",", with: "-")
}

func reminderTermHashtag(from reminder: EKReminder) -> String? {
    guard let notes = reminder.notes else { return nil }
    let lines = notes.components(separatedBy: .newlines)
    guard let range = metadataRange(in: lines) else { return nil }
    return lines[range]
        .joined(separator: "\n")
        .components(separatedBy: .whitespacesAndNewlines)
        .first(where: { value in
            let parts = value.split(separator: "-", maxSplits: 1)
            guard parts.count == 2, parts[0].hasPrefix("#"), Int(parts[0].dropFirst()) != nil else {
                return false
            }
            return parts[1] == "1" || parts[1] == "2" || parts[1] == "여름학기" || parts[1] == "겨울학기"
        })
}

func reminderTermValueFromURL(_ reminder: EKReminder) -> String? {
    guard let url = reminder.url,
          let components = URLComponents(url: url, resolvingAgainstBaseURL: false) else {
        return nil
    }
    return components.queryItems?.first(where: { $0.name == "selectYearhakgi" })?.value
}

func matchesLegacyIdentity(_ reminder: EKReminder, assignment: Assignment) -> Bool {
    guard reminderCourse(from: reminder) == assignment.course else {
        return false
    }
    if let hashtag = termHashtag(for: assignment.termValue), reminderTermHashtag(from: reminder) == hashtag {
        return true
    }
    return reminderTermValueFromURL(reminder) == assignment.termValue
}

func couldBeUnverifiedLegacyMatch(_ reminder: EKReminder, assignment: Assignment) -> Bool {
    if let course = reminderCourse(from: reminder), course != assignment.course {
        return false
    }
    if let hashtag = reminderTermHashtag(from: reminder),
       let expected = termHashtag(for: assignment.termValue), hashtag != expected {
        return false
    }
    if let termValue = reminderTermValueFromURL(reminder), termValue != assignment.termValue {
        return false
    }
    return true
}

func uniqueMatch(_ matches: [KnownReminder]) -> KnownReminder? {
    return matches.count == 1 ? matches[0] : nil
}

func legacyMatch(for assignment: Assignment, in known: [KnownReminder]) -> KnownReminder? {
    let resourceParts = Set((assignment.legacyIds ?? []).compactMap(legacyResourcePart))
    guard !resourceParts.isEmpty else { return nil }
    let matches = known.filter { candidate in
        guard let resourcePart = legacyResourcePart(from: candidate.id),
              resourceParts.contains(resourcePart),
              matchesLegacyIdentity(candidate.reminder, assignment: assignment) else {
            return false
        }
        return true
    }
    return uniqueMatch(matches)
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
let known = existingReminders().compactMap { reminder -> KnownReminder? in
    guard let id = assignmentID(from: reminder) else { return nil }
    return KnownReminder(id: id, reminder: reminder)
}

var result = SyncResult()
for assignment in request.assignments {
    guard let dueAt = assignment.dueAt else {
        result.skipped += 1
        continue
    }

    let hasKnownSource = assignment.knownSourceHash?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
    let stableMatch = uniqueMatch(known.filter { $0.id == assignment.id })
    let legacyIds = Set(assignment.legacyIds ?? [])
    let legacyMatchById = uniqueMatch(known.filter {
        legacyIds.contains($0.id) && matchesLegacyIdentity($0.reminder, assignment: assignment)
    })
    let exactMatch = stableMatch ?? legacyMatchById
    let exactMatchedId = exactMatch?.id
    let fallbackMatch = exactMatchedId == nil ? legacyMatch(for: assignment, in: known) : nil
    let matchedId = exactMatchedId ?? fallbackMatch?.id
    let matchedReminder = exactMatch?.reminder ?? fallbackMatch?.reminder
    let requestedLegacyIds = Set(assignment.legacyIds ?? [])
    let requestedResourceParts = Set(requestedLegacyIds.compactMap(legacyResourcePart))
    let hasUnverifiedLegacyCandidate = known.contains { candidate in
        let exactCandidate = requestedLegacyIds.contains(candidate.id)
        let resourceCandidate = legacyResourcePart(from: candidate.id).map(requestedResourceParts.contains) == true
        return (exactCandidate || resourceCandidate) && couldBeUnverifiedLegacyMatch(candidate.reminder, assignment: assignment)
    }
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

    if hasUnverifiedLegacyCandidate {
        result.skipped += 1
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

let data = try encodeBridgeResponse(result)
FileHandle.standardOutput.write(data)
