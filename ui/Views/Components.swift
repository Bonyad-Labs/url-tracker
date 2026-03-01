import SwiftUI

struct WhitelistRow: View {
    let item: WhitelistItem
    
    var dateString: String {
        let date = Date(timeIntervalSince1970: TimeInterval(item.timestamp))
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .none
        return formatter.string(from: date)
    }
    
    var body: some View {
        HStack(spacing: 0) {
            // Type
            HStack {
                Image(systemName: item.type == "url" ? "link" : "globe")
                    .foregroundColor(.blue)
                Text(item.type.capitalized)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .frame(width: 80, alignment: .leading)
            
            // Value
            Text(item.value)
                .font(.system(.body, design: .monospaced))
                .lineLimit(1)
                .truncationMode(.middle)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 8)
            
            // Date
            Text(dateString)
                .font(.caption)
                .foregroundColor(.secondary)
                .frame(width: 80, alignment: .leading)
            
            // Action
            Button(action: {
                print("DELETE_WHITELIST|\(item.value)")
                fflush(stdout)
            }) {
                Image(systemName: "trash")
                    .font(.caption)
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 8)
            .foregroundColor(.red.opacity(0.8))
            .onHover { inside in
                if inside { NSCursor.pointingHand.set() } else { NSCursor.arrow.set() }
            }
        }
        .padding(.vertical, 8)
        .padding(.horizontal, 4)
    }
}

struct SearchRow: View {
    let entry: SearchEntry
    let isSelected: Bool
    
    var faviconURL: URL? {
        if let domain = URL(string: entry.url)?.host {
            return URL(string: "https://www.google.com/s2/favicons?domain=\(domain)&sz=64")
        }
        return nil
    }
    
    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            AsyncImage(url: faviconURL) { image in
                image.resizable()
            } placeholder: {
                Image(systemName: "globe")
                    .foregroundColor(.secondary)
            }
            .frame(width: 24, height: 24)
            .cornerRadius(4)
            .padding(.top, 2)
            
            VStack(alignment: .leading, spacing: 4) {
                Text(entry.title.isEmpty ? "No Title" : entry.title)
                    .font(.subheadline)
                    .fontWeight(.bold)
                    .lineLimit(1)
                    .foregroundColor(isSelected ? .white : .primary)
                
                if !entry.description.isEmpty {
                    Text(entry.description)
                        .font(.caption)
                        .foregroundColor(isSelected ? .white.opacity(0.7) : .secondary)
                        .lineLimit(2)
                        .truncationMode(.tail)
                }
                
                HStack {
                    Text(entry.url)
                        .font(.system(size: 10))
                        .foregroundColor(isSelected ? .white.opacity(0.6) : .secondary.opacity(0.8))
                        .lineLimit(1)
                        .truncationMode(.middle)
                    
                    if !entry.category.isEmpty {
                        Spacer()
                        Text(entry.category)
                            .font(.system(size: 9, weight: .bold))
                            .padding(.horizontal, 4)
                            .padding(.vertical, 1)
                            .background(isSelected ? Color.white.opacity(0.2) : Color.blue.opacity(0.1))
                            .foregroundColor(isSelected ? .white : .blue)
                            .cornerRadius(3)
                    }
                }
            }
        }
        .padding(.vertical, 6)
    }
}

struct SidebarRow: View {
    let title: String
    let icon: String
    let selection: SidebarSelection
    @Binding var currentSelection: SidebarSelection
    let count: Int
    
    var isSelected: Bool { selection == currentSelection }
    
    var body: some View {
        Button(action: { currentSelection = selection }) {
            HStack {
                Label(title, systemImage: icon)
                Spacer()
                if count > 0 {
                    Text("\(count)")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(isSelected ? Color.white.opacity(0.3) : Color.secondary.opacity(0.1))
                        .cornerRadius(10)
                }
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .foregroundColor(isSelected ? .white : .primary)
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(isSelected ? Color.blue : Color.clear)
        .cornerRadius(6)
    }
}

struct FlowLayout: View {
    let spacing: CGFloat
    let items: [String]
    let content: (String) -> AnyView
    
    @State private var totalHeight = CGFloat.zero

    var body: some View {
        VStack {
            GeometryReader { geometry in
                self.generateContent(in: geometry)
            }
        }
        .frame(height: totalHeight)
    }

    private func generateContent(in g: GeometryProxy) -> some View {
        var width = CGFloat.zero
        var height = CGFloat.zero

        return ZStack(alignment: .topLeading) {
            ForEach(items, id: \.self) { item in
                self.content(item)
                    .padding([.horizontal, .vertical], spacing)
                    .alignmentGuide(.leading, computeValue: { d in
                        if (abs(width - d.width) > g.size.width) {
                            width = 0
                            height -= d.height
                        }
                        let result = width
                        if item == self.items.last! {
                            width = 0 // last item
                        } else {
                            width -= d.width
                        }
                        return result
                    })
                    .alignmentGuide(.top, computeValue: { d in
                        let result = height
                        if item == self.items.last! {
                            height = 0 // last item
                        }
                        return result
                    })
            }
        }.background(viewHeightReader($totalHeight))
    }

    private func viewHeightReader(_ binding: Binding<CGFloat>) -> some View {
        return GeometryReader { geometry -> Color in
            let rect = geometry.frame(in: .local)
            DispatchQueue.main.async {
                binding.wrappedValue = rect.size.height
            }
            return .clear
        }
    }
}
