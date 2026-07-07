import { Link } from "react-router-dom";

// rendered by "*" catch-all route for any unknown URL
export default function NotFound() {
	return (
	<div className="mx-auto max-w-2xl px-4 py-16 text-center">
		<h1 className="text-3xl font-bold text-foreground">404 - Page not found</h1>
		<p className="mt-2 text-muted">That page doesn't exist or has moved</p>
		<Link to="/" className="mt-4 inline-block text-accent hover:underline">
			Back to home
		</Link>
	</div>
	);
}
