// src/pages/Landing.tsx
import React from "react";
import { Link } from "react-router-dom";
import Header from "../components/Header";
import Footer from "../components/Footer";
import AnimatedSection from "../components/AnimatedSection";

const Landing: React.FC = () => {
  return (
    <div className="bg-base-100 text-base-content">
      <Header />

      {/* Hero Section */}
      <section className="hero min-h-screen bg-base-200">
        <div className="hero-content text-center">
          <div className="max-w-md">
            <h2 className="text-5xl font-bold">
              O Futuro da Edição LaTeX é Colaborativo
            </h2>
            <p className="py-6">
              Crie, edite e compile seus documentos LaTeX em tempo real, com
              controle de versão Git integrado e colaboração simplificada.
            </p>
            <Link to="/register" className="btn btn-primary">
              Comece Agora
            </Link>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <AnimatedSection className="container mx-auto px-4 py-20">
        <h3 className="text-4xl font-bold text-center mb-12">
          Recursos Incríveis
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {/* Feature 1 */}
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <h4 className="card-title">Compilação Rápida e Eficiente</h4>
              <p>
                Com um único clique, veja as alterações no seu PDF, sem a
                necessidade de compilar manualmente.
              </p>
            </div>
          </div>
          {/* Feature 2 */}
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <h4 className="card-title">Controle de Versão com Git</h4>
              <p>
                Cada projeto é um repositório Git, permitindo que você versione
                seu trabalho de forma eficiente.
              </p>
            </div>
          </div>
          {/* Feature 3 */}
          <div className="card bg-base-100 shadow-xl">
            <div className="card-body">
              <h4 className="card-title">
                Gerenciamento de Múltiplos Projetos
              </h4>
              <p>
                Organize seus documentos em múltiplos projetos, facilitando o
                gerenciamento e a organização do seu trabalho.
              </p>
            </div>
          </div>
        </div>
      </AnimatedSection>

      {/* Testimonials Section */}
      <div className="bg-base-200">
        <AnimatedSection className="container mx-auto px-4 py-20">
          <h3 className="text-4xl font-bold text-center mb-12">
            O Que Nossos Usuários Dizem
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {/* Testimonial 1 */}
            <div className="card bg-base-100 shadow-xl">
              <div className="card-body">
                <p>
                  "O Superfolha revolucionou a forma como escrevo meus artigos.
                  A compilação em tempo real é mágica!"
                </p>
                <h5 className="font-bold mt-4">
                  - Dr. Ana Souza, Pesquisadora
                </h5>
              </div>
            </div>
            {/* Testimonial 2 */}
            <div className="card bg-base-100 shadow-xl">
              <div className="card-body">
                <p>
                  "A melhor ferramenta para colaborar em documentos LaTeX que já
                  usei. A integração com Git é perfeita."
                </p>
                <h5 className="font-bold mt-4">
                  - João Silva, Estudante de Doutorado
                </h5>
              </div>
            </div>
            {/* Testimonial 3 */}
            <div className="card bg-base-100 shadow-xl">
              <div className="card-body">
                <p>
                  "Finalmente uma solução moderna para LaTeX. A interface é
                  limpa, intuitiva e muito poderosa."
                </p>
                <h5 className="font-bold mt-4">
                  - Maria Oliveira, Desenvolvedora
                </h5>
              </div>
            </div>
          </div>
        </AnimatedSection>
      </div>

      {/* Final CTA Section */}
      <AnimatedSection className="container mx-auto px-4 py-20 text-center">
        <h3 className="text-4xl font-bold mb-4">
          Pronto para Otimizar Seu Fluxo de Trabalho?
        </h3>
        <p className="text-xl mb-8">
          Junte-se a milhares de desenvolvedores e acadêmicos que já estão
          usando o Superfolha.
        </p>
        <Link to="/register" className="btn btn-primary">
          Crie Sua Conta Gratuita
        </Link>
      </AnimatedSection>

      {/* Footer */}
      <Footer />
    </div>
  );
};

export default Landing;
