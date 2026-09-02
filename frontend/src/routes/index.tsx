import { BrowserRouter, Routes, Route } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Home from "../pages/Home";
import AddListing from "../pages/AddListing";
import ListingDetail from "../pages/ListingDetail";
import Search from "../pages/Search";
import Profile from "../pages/Profile";
import User from "../pages/User";
import Dashboard from "../pages/Dashboard";
import Orders from "../pages/Orders";
import OrderDetail from "../pages/OrderDetail";
import Notifications from "../pages/Notifications";
import Privacy from "../pages/Privacy";
import Terms from "../pages/Terms";
import NotFound from "../pages/NotFound";
import AuthCallback from "../pages/AuthCallback";

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Top-level redirect target from the OAuth flow - rendered outside the
            app shell since it's a transient page, not normal UI. */}
        <Route path="/auth/callback" element={<AuthCallback />} />
        <Route element={<Layout />}>
          <Route path="/" element={<Home />} />
          {/* ":id" is a URL parameter - the page reads it with useParams() */}
          <Route path="/listings/:id" element={<ListingDetail />} />
          <Route path="/addlisting" element={<AddListing />} />
          <Route path="/search" element={<Search />} />
          <Route path="/profile" element={<Profile />} />
          <Route path="/users/:id" element={<User />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/orders/:id" element={<OrderDetail />} />
          <Route path="/notifications" element={<Notifications />} />
          <Route path="/privacy" element={<Privacy />} />
          <Route path="/terms" element={<Terms />} />
          {/* any unmatched URL renders the 404 inside the shell */}
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
