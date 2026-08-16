import { BrowserRouter, Routes, Route } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Home from "../pages/Home";
import ListingDetail from "../pages/ListingDetail";
import Search from "../pages/Search";
import Profile from "../pages/Profile";
import User from "../pages/User";
import Dashboard from "../pages/Dashboard";
import Notifications from "../pages/Notifications";
import Privacy from "../pages/Privacy";
import Terms from "../pages/Terms";
import NotFound from "../pages/NotFound";

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Home />} />
          {/* ":id" is a URL parameter - the page reads it with useParams() */}
          <Route path="/listings/:id" element={<ListingDetail />} />
          <Route path="/search" element={<Search />} />
          <Route path="/profile" element={<Profile />} />
          <Route path="/user" element={<User />} />
          {/*
            /user is just a placeholder for now, we will replace with profileId
            <Route path="/profile/:profileId" element={<User />} />
          */}
          <Route path="/dashboard" element={<Dashboard />} />
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
