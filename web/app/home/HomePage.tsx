"use client";

// Top Ribbon
import TopHeader from "../sections/header/TopHeader";

// Main Header
import GlobalHeader from "../sections/header/GlobalHeader";

// Sections
import HeroSection from "../sections/hero/HeroSection";
import CoreCapabilities from "../sections/capabilities/CoreCapabilities";
import OperationalRail from "../sections/operational/OperationalRail";
import PlatformOverview from "../sections/platform/Overview";
import MissionSection from "../sections/mission/MissionSection";
import TechnologySovereignty from "../sections/sovereignty/TechnologySovereignty";
import StrategicRoadmap from "../sections/roadmap/StrategicRoadmap";
import ContactSection from "../sections/contact/ContactSection";
import Footer from "../sections/footer/Footer";
import "../styles/HomePage.css";

export default function HomePage() {
  return (
    <main className="home-page" id="top">

      <TopHeader />
      <GlobalHeader />

      <HeroSection />
      <OperationalRail />
      <PlatformOverview />
      <MissionSection />
      <CoreCapabilities />
      <TechnologySovereignty />
      <StrategicRoadmap />
      <ContactSection />

      <Footer />
    </main>
  );
}
