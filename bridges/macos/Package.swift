// swift-tools-version: 6.0

import PackageDescription
import Foundation

// Embed usage descriptions in standalone tools; no .app bundle is distributed.
let packageRoot = URL(fileURLWithPath: #filePath).deletingLastPathComponent()
func bridgeLinkerSettings(_ name: String) -> [LinkerSetting] {
    let info = packageRoot.appendingPathComponent("Info").appendingPathComponent(name + ".plist").path
    return [.unsafeFlags(["-Xlinker", "-sectcreate", "-Xlinker", "__TEXT", "-Xlinker", "__info_plist", "-Xlinker", info])]
}

let package = Package(
    name: "KLAPMacOSBridges",
    platforms: [
        .macOS("12")
    ],
    products: [
        .executable(name: "TranscriptBridge", targets: ["TranscriptBridge"]),
        .executable(name: "ReminderBridge", targets: ["ReminderBridge"]),
        .executable(name: "CalendarBridge", targets: ["CalendarBridge"]),
        .executable(name: "CategoryBridge", targets: ["CategoryBridge"])
    ],
    targets: [
        .target(name: "SpeechCore"),
        .executableTarget(name: "TranscriptBridge", dependencies: ["SpeechCore"], linkerSettings: bridgeLinkerSettings("TranscriptBridge")),
        .testTarget(name: "SpeechCoreTests", dependencies: ["SpeechCore"]),
        // Preserve the distributed EventKit scripts' Swift 5 language mode.
        .target(name: "EventKitCore", swiftSettings: [.swiftLanguageMode(.v5)]),
        .executableTarget(name: "ReminderBridge", dependencies: ["EventKitCore"], swiftSettings: [.swiftLanguageMode(.v5)], linkerSettings: bridgeLinkerSettings("ReminderBridge")),
        .executableTarget(name: "CalendarBridge", dependencies: ["EventKitCore"], swiftSettings: [.swiftLanguageMode(.v5)], linkerSettings: bridgeLinkerSettings("CalendarBridge")),
        .executableTarget(name: "CategoryBridge", dependencies: ["EventKitCore"], swiftSettings: [.swiftLanguageMode(.v5)], linkerSettings: bridgeLinkerSettings("CategoryBridge")),
        .testTarget(name: "EventKitCoreTests", dependencies: ["EventKitCore"], swiftSettings: [.swiftLanguageMode(.v5)])
    ]
)
