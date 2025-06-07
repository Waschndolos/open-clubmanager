import { Link } from 'react-router-dom'

export default function NotFound() {
    return (
        <div className="flex flex-col items-center justify-center h-screen text-center">
            <h1 className="text-5xl font-bold text-red-500 mb-4">404</h1>
            <p className="text-xl mb-2">Seite nicht gefunden</p>
            <Link to="/dashboard" className="mt-4 text-blue-500 underline">
                Zurück zur Startseite
            </Link>
        </div>
    )
}
