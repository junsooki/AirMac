import SwiftUI

struct RemoteDisplayView: View {
    @ObservedObject var viewModel: ConnectionViewModel
    @State private var showKeyboard = false

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()

            if let frame = viewModel.currentFrame {
                GeometryReader { geo in
                    let displaySize = aspectFitSize(
                        imageSize: CGSize(width: viewModel.frameWidth, height: viewModel.frameHeight),
                        containerSize: geo.size
                    )
                    let offsetX = (geo.size.width - displaySize.width) / 2
                    let offsetY = (geo.size.height - displaySize.height) / 2

                    ZStack {
                        Image(uiImage: frame)
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .frame(width: displaySize.width, height: displaySize.height)
                            .position(x: geo.size.width / 2, y: geo.size.height / 2)

                        TouchInputView(
                            viewModel: viewModel,
                            displaySize: displaySize,
                            displayOffset: CGPoint(x: offsetX, y: offsetY),
                            frameSize: CGSize(width: viewModel.frameWidth, height: viewModel.frameHeight)
                        )
                        .frame(width: displaySize.width, height: displaySize.height)
                        .position(x: geo.size.width / 2, y: geo.size.height / 2)
                    }
                }
            } else {
                VStack(spacing: 12) {
                    ProgressView()
                        .tint(.white)
                    Text("Waiting for frames...")
                        .foregroundColor(.gray)
                }
            }

            // Status overlay
            VStack {
                HStack {
                    statusIndicator
                    Spacer()
                    Button {
                        showKeyboard.toggle()
                    } label: {
                        Image(systemName: "keyboard")
                            .foregroundColor(.white)
                            .padding(8)
                            .background(.ultraThinMaterial)
                            .clipShape(Circle())
                    }
                    Button {
                        viewModel.disconnect()
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundColor(.white)
                            .padding(8)
                            .background(.ultraThinMaterial)
                            .clipShape(Circle())
                    }
                }
                .padding(.horizontal)
                .padding(.top, 4)
                Spacer()
            }
        }
        .ignoresSafeArea()
        .navigationBarHidden(true)
        .statusBarHidden(true)
        .sheet(isPresented: $showKeyboard) {
            VirtualKeyboardView(viewModel: viewModel)
                .presentationDetents([.medium])
        }
    }

    private var statusIndicator: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(viewModel.state == .connected ? Color.green : Color.orange)
                .frame(width: 8, height: 8)
            Text(viewModel.connectedHostID ?? "")
                .font(.caption)
                .foregroundColor(.white)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(.ultraThinMaterial)
        .clipShape(Capsule())
    }

    /// Compute aspect-fit display size (mirrors ebiten.go:Draw scale calculation)
    private func aspectFitSize(imageSize: CGSize, containerSize: CGSize) -> CGSize {
        guard imageSize.width > 0, imageSize.height > 0 else { return .zero }
        let scaleX = containerSize.width / imageSize.width
        let scaleY = containerSize.height / imageSize.height
        let scale = min(scaleX, scaleY)
        return CGSize(
            width: imageSize.width * scale,
            height: imageSize.height * scale
        )
    }
}
