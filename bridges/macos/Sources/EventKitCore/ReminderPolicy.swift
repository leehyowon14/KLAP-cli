import EventKit
import Foundation

public func reminderNotes(_ notes: String) -> String {
    notes.components(separatedBy: .newlines).filter { !$0.hasPrefix("마감: ") }.joined(separator: "\n")
}

public func applyReminderDueDate(_ due: Date, to reminder: EKReminder, calendar: Calendar = .current) {
    reminder.dueDateComponents = calendar.dateComponents([.year, .month, .day, .hour, .minute], from: due)
    reminder.startDateComponents = nil
}

// The shared default is 24 hours; explicit CLI settings remain effective.
public func applyReminderAlarm(_ due: Date, beforeMinutes: Int, to reminder: EKReminder,
                               now: Date = Date(), calendar: Calendar = .current) {
    guard let alarm = calendar.date(byAdding: .minute, value: -max(beforeMinutes, 1), to: due), alarm > now else {
        reminder.alarms = nil
        return
    }
    reminder.alarms = [EKAlarm(absoluteDate: alarm)]
}
