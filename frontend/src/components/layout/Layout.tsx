// Outlet is the placeholder where the matched child route renders
import { Outlet } from "react-router-dom";
import Header from "./Header";
import Footer from "./Footer";

export default function Layout() {
	return (
		<div className="flex min-h-screen flex-col bg-surface text-foreground">
			<Header />
			<main className="flex-1">
				<Outlet />
			</main>
			<Footer />
		</div>
	);
}
