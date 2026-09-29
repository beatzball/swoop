// swift-tools-version: 5.9
// The macOS frame: the only per-OS code in swoop. A floating panel, a global
// hotkey, and a libghostty terminal surface that runs `swoop`. It holds no
// launcher logic; see the "Design: the per-OS frame" issue.
import PackageDescription

let package = Package(
    name: "swoop-shell-mac",
    platforms: [.macOS(.v13)],
    dependencies: [
        // libghostty prebuilt as a Swift package, Metal rendering included,
        // tracking Ghostty's main branch. Its .exec backend spawns the
        // command and reports when it exits, which is all the frame needs.
        .package(url: "https://github.com/Lakr233/libghostty-spm.git", from: "1.6.0"),
    ],
    targets: [
        .executableTarget(
            name: "swoop-shell-mac",
            dependencies: [
                .product(name: "GhosttyTerminal", package: "libghostty-spm"),
            ],
            // The menu bar mark, the owl as a one-colour template image at
            // 18 and 36 px. It rides in the target's resource bundle beside
            // the executable, like libghostty's.
            resources: [.copy("Resources")],
            linkerSettings: [
                // Carbon for RegisterEventHotKey: a global hotkey with no
                // Accessibility permission and no event tap.
                .linkedFramework("Carbon"),
            ]
        ),
    ]
)
