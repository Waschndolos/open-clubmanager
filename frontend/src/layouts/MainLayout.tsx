import { useState } from 'react'
import { Outlet, Link, useLocation } from 'react-router-dom'
import { Menu, X } from 'lucide-react'

export default function MainLayout() {
    const [open, setOpen] = useState(false)
    const location = useLocation()

    return (
        <div className="flex h-screen">

            <aside
                className={`fixed z-40 inset-y-0 left-0 w-64 bg-gray-900 text-white p-4 transform transition-transform duration-200 ease-in-out
        ${open ? 'translate-x-0' : '-translate-x-full'} 
        md:translate-x-0 md:static md:block`}
            >
                <nav className="flex flex-col space-y-4 mt-4">
                    <NavLink to="/" label="🏠 Home" currentPath={location.pathname} />
                </nav>
            </aside>

            {/* Overlay on mobile */}
            {open && (
                <div
                    className="fixed inset-0 bg-black bg-opacity-30 z-30 md:hidden"
                    onClick={() => setOpen(false)}
                />
            )}

            {/* Main content area */}
            <div className="flex flex-col flex-1 md:ml-64">
                {/* Header */}
                <header className="bg-blue-600 text-white p-4 flex items-center justify-between shadow">
                    <h1 className="text-xl font-bold">Meine App</h1>
                    <button className="md:hidden" onClick={() => setOpen(!open)}>
                        {open ? <X size={24} /> : <Menu size={24} />}
                    </button>
                </header>

                {/* Page Content */}
                <main className="flex-1 p-6 overflow-y-auto">
                    <Outlet />
                </main>
            </div>
        </div>
    )
}

function NavLink({ to, label, currentPath }: { to: string; label: string; currentPath: string }) {
    const active = currentPath === to
    return (
        <Link
            to={to}
            className={`px-3 py-2 rounded transition ${
                active ? 'bg-blue-500 text-white font-bold' : 'hover:bg-gray-700'
            }`}
        >
            {label}
        </Link>
    )
}
