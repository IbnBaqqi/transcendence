import { BrowserRouter, Routes, Route } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Home from "../pages/Home";
import Privacy from "../pages/Privacy";
import Terms from "../pages/Terms";

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/privacy" element={<Home />} />
        <Route path="/terms" element={<Home />} />
      </Routes>
    </BrowserRouter>
  );
}
