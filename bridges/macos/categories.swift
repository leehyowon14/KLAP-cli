import EventKit
import Foundation

struct CategoryOptions: Codable {
    let reminders: [String]
    let calendars: [String]
}

let store = EKEventStore()
let semaphore = DispatchSemaphore(value: 0)
let group = DispatchGroup()
var reminderGranted = false
var calendarGranted = false
var permissionError: Error?

group.enter()
if #available(macOS 14.0, *) {
    store.requestFullAccessToReminders { ok, error in
        reminderGranted = ok
        if permissionError == nil {
            permissionError = error
        }
        group.leave()
    }
} else {
    store.requestAccess(to: .reminder) { ok, error in
        reminderGranted = ok
        if permissionError == nil {
            permissionError = error
        }
        group.leave()
    }
}

group.enter()
if #available(macOS 14.0, *) {
    store.requestFullAccessToEvents { ok, error in
        calendarGranted = ok
        if permissionError == nil {
            permissionError = error
        }
        group.leave()
    }
} else {
    store.requestAccess(to: .event) { ok, error in
        calendarGranted = ok
        if permissionError == nil {
            permissionError = error
        }
        group.leave()
    }
}

group.notify(queue: .global()) {
    semaphore.signal()
}
semaphore.wait()

if let permissionError {
    throw permissionError
}

let reminders = reminderGranted ? store.calendars(for: .reminder).map(\.title).sorted() : []
let calendars = calendarGranted ? store.calendars(for: .event).filter { $0.allowsContentModifications }.map(\.title).sorted() : []
let result = CategoryOptions(reminders: reminders, calendars: calendars)
let data = try JSONEncoder().encode(result)
FileHandle.standardOutput.write(data)
