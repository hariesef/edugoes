import './Sidebar.css'

type SidebarItem = {
  key: string
  label: string
}

export function Sidebar({
  items,
  active,
  onSelect,
}: {
  items: SidebarItem[]
  active: string
  onSelect: (key: string) => void
}) {
  return (
    <aside className="sidebar">
      <div className="sidebar-title">LTI PoC</div>
      <nav>
        {items.map((item) => (
          <button
            key={item.key}
            className={`menu-item ${active === item.key ? 'active' : ''}`}
            onClick={() => onSelect(item.key)}
          >
            {item.label}
          </button>
        ))}
      </nav>
    </aside>
  )
}
