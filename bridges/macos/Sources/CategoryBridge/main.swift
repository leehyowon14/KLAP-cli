import EventKit
import EventKitCore
import Foundation

struct CategoryOptions: Codable {
    let reminders: [String]
    let calendars: [String]
}

let store = EKEventStore()
let granted = try requestPermissions([.reminder, .event], from: store)
let reminderGranted = granted[0]
let calendarGranted = granted[1]

let reminders = reminderGranted ? store.calendars(for: .reminder).map(\.title).sorted() : []
let calendars = calendarGranted ? store.calendars(for: .event).filter { $0.allowsContentModifications }.map(\.title).sorted() : []
let result = CategoryOptions(reminders: reminders, calendars: calendars)
let data = try encodeBridgeResponse(result)
FileHandle.standardOutput.write(data)
