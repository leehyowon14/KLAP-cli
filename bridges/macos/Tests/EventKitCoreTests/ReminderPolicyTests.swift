import EventKit
import Foundation
import Testing
@testable import EventKitCore

struct ReminderPolicyTests {
    @Test func datesAndAlarmsDoNotWriteToStore() {
        let reminder = EKReminder(eventStore: EKEventStore())
        let now = Date(timeIntervalSince1970: 1_800_000_000)
        let due = now.addingTimeInterval(3 * 86400)
        reminder.startDateComponents = DateComponents(year: 2020, month: 1, day: 1)
        applyReminderDueDate(due, to: reminder)
        #expect(reminder.startDateComponents == nil)
        #expect(reminder.dueDateComponents?.day == Calendar.current.component(.day, from: due))
        for minutes in [1, 60, 1440, 0, -1] {
            applyReminderAlarm(due, beforeMinutes: minutes, to: reminder, now: now)
            #expect(reminder.alarms?.first?.absoluteDate == due.addingTimeInterval(-Double(max(minutes, 1)) * 60))
        }
        for seconds in [-60.0, 0, 3600, 86400] {
            applyReminderAlarm(now.addingTimeInterval(seconds), beforeMinutes: 1440, to: reminder, now: now)
            #expect(reminder.alarms?.isEmpty != false)
        }
    }

    @Test func removesOnlyRedundantDeadlineLines() {
        #expect(reminderNotes("과제\n마감: 내일\nID: 123") == "과제\nID: 123")
        #expect(reminderNotes("마감: 내일\r\n본문의 마감: 설명") == "\n본문의 마감: 설명")
        #expect(reminderNotes("") == "")
    }
}
