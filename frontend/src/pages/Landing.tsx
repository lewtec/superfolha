// src/pages/Landing.tsx
import React from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import Header from "../components/Header";
import Footer from "../components/Footer";
import AnimatedSection from "../components/AnimatedSection";
import { BrandMark } from "../components/Brand";

const Landing: React.FC = () => {
  const { t } = useTranslation("landing");

  return (
    <div className="min-h-dvh flex flex-col bg-base-100 text-base-content">
      <Header />

      {/* Hero Section */}
      <section className="hero min-h-[calc(100dvh-var(--shell-height))] flex-1 bg-base-200">
        <div className="hero-content text-center">
          <div className="max-w-md">
            <BrandMark className="mx-auto mb-6 h-24 w-24" alt="" />
            <h2 className="text-5xl font-bold">{t("landing_title")}</h2>
            <p className="py-6">{t("landing_subtitle")}</p>
            <Link to="/register" className="btn btn-primary">
              {t("landing_cta")}
            </Link>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <AnimatedSection className="container mx-auto px-4 py-20">
        <h3 className="text-4xl font-bold text-center mb-12">
          {t("features_title")}
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {/* Feature 1 */}
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <h4 className="card-title">{t("feature1_title")}</h4>
              <p>{t("feature1_description")}</p>
            </div>
          </div>
          {/* Feature 2 */}
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <h4 className="card-title">{t("feature2_title")}</h4>
              <p>{t("feature2_description")}</p>
            </div>
          </div>
          {/* Feature 3 */}
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <h4 className="card-title">{t("feature3_title")}</h4>
              <p>{t("feature3_description")}</p>
            </div>
          </div>
        </div>
      </AnimatedSection>

      {/* Testimonials Section */}
      <div className="bg-base-200">
        <AnimatedSection className="container mx-auto px-4 py-20">
          <h3 className="text-4xl font-bold text-center mb-12">
            {t("testimonials_title")}
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {/* Testimonial 1 */}
            <div className="card bg-base-100 shadow-xl">
              <div className="card-body">
                <p>"{t("testimonial1_text")}"</p>
                <h5 className="font-bold mt-4">{t("testimonial1_author")}</h5>
              </div>
            </div>
            {/* Testimonial 2 */}
            <div className="card bg-base-100 shadow-xl">
              <div className="card-body">
                <p>"{t("testimonial2_text")}"</p>
                <h5 className="font-bold mt-4">{t("testimonial2_author")}</h5>
              </div>
            </div>
            {/* Testimonial 3 */}
            <div className="card bg-base-100 shadow-xl">
              <div className="card-body">
                <p>"{t("testimonial3_text")}"</p>
                <h5 className="font-bold mt-4">{t("testimonial3_author")}</h5>
              </div>
            </div>
          </div>
        </AnimatedSection>
      </div>

      {/* Final CTA Section */}
      <AnimatedSection className="container mx-auto px-4 py-20 text-center">
        <h3 className="text-4xl font-bold mb-4">{t("final_cta_title")}</h3>
        <p className="text-xl mb-8">{t("final_cta_subtitle")}</p>
        <Link to="/register" className="btn btn-primary">
          {t("final_cta_button")}
        </Link>
      </AnimatedSection>

      {/* Footer */}
      <Footer />
    </div>
  );
};

export default Landing;
