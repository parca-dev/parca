describe('Parca Profiler', () => {
  // it('should capture a screenshot', () => {
  //   cy.visit('http://localhost:3000/');
  //   cy.wait(1000);
  //   cy.screenshot('profiler-no-query', {overwrite: true});
  // });
  // it('should capture a screenshot after query made', () => {
  //   cy.visit('http://localhost:3000/');
  //   cy.get('#listbox-button').click();
  //   cy.get('#listbox-option-0').click();
  //   cy.get('#search-button').click();
  //   // Wait for the metrics graph to finish rendering
  //   cy.get('#profile-metrics-graph').should('be.visible');
  //   // Take a screenshot of the viewport
  //   cy.screenshot('profiler-with-query', {overwrite: true});
  // });

  // it('should capture a screenshot of compare mode', () => {
  //   cy.visit('http://localhost:3000/');
  //   cy.get('#listbox-button').click();
  //   cy.get('#listbox-option-0').click();
  //   cy.get('#search-button').click();
  //   cy.get('#compare-button').click();
  //   // Wait for the page to finish rendering
  //   cy.get('#profile-metrics-graph').should('be.visible');
  //   cy.screenshot('profiler-compare-mode', {overwrite: true});
  // });
  it('should capture a screenshot of the icicle graph only', () => {
    cy.visit('http://localhost:3000/');
    cy.get('#listbox-button').click();
    cy.get('#listbox-option-0').should('be.visible').click();
    cy.get('#search-button').click();
    // Move the mouse to a specific position over the specified element
    cy.get('#profile-metrics-graph').should('be.visible').click();
    cy.get('#profile-metrics-graph').should('be.visible').click();
    // Wait for the icicle graph to load
    const icicleGraph = cy.get('#profile-icicle-graph').should('be.visible');
    // Move the mouse outside the graph to prevent the tooltip from showing
    cy.get('body').trigger('mousemove', 0, 0);
    // Take a screenshot of the icicle graph
    icicleGraph.screenshot('icicle-graph', {overwrite: true});
    // Take a screenshot of the profile view
    cy.get('#profile-view').screenshot('profile-view', {overwrite: true});
  });
  it('should capture a screenshot of the compare mode profile view', () => {
    cy.visit('http://localhost:3000/');
    cy.get('#listbox-button').click();
    cy.get('#listbox-option-0').click();
    cy.get('#search-button').click();
    cy.get('#compare-button').click();
    // Wait for the page to finish rendering
    const metricsGraphs = cy.get('#profile-metrics-graph').should('have.length.at.least', 2);

    metricsGraphs.then(elements => {
      // Do something with the elements
      elements.each((index, element) => {
        // Move the mouse to a specific position over the specified element
        cy.wrap(element).click();
        cy.wrap(element).click();
      });
    });

    // Wait for the icicle graph to load
    cy.get('#profile-icicle-graph').should('be.visible');
    // Move the mouse outside the graph to prevent the tooltip from showing
    cy.get('body').trigger('mousemove', 0, 0);

    // Capture screenshot of profile view
    cy.get('#profile-view').screenshot('profile-view', {overwrite: true});
  });
});
