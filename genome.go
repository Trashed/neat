// genome.go implementation of the genome in evolution.
//
// Copyright (C) 2017  Jin Yeom
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package neat

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"
)

var (
	Innovation int
)

// NodeGene is an implementation of each node in the graph representation of a
// genome. Each node consists of a node ID, its type, and the activation type.
type NodeGene struct {
	ID         int             `json:"id"`         // node ID
	Type       string          `json:"type"`       // node type
	Activation *ActivationFunc `json:"activation"` // activation function
}

// NewNodeGene returns a new instance of NodeGene, given its ID, its type, and
// the activation function of this node.
func NewNodeGene(id int, ntype string, activation *ActivationFunc) *NodeGene {
	return &NodeGene{id, ntype, activation}
}

// Copy returns a deep copy of this node gene.
func (n *NodeGene) Copy() *NodeGene {
	return &NodeGene{n.ID, n.Type, n.Activation}
}

// String returns a string representation of the node.
func (n *NodeGene) String() string {
	return fmt.Sprintf("[%s(%d, %s)]", n.Type, n.ID, n.Activation.Name)
}

// ConnGene is an implementation of a connection between two nodes in the graph
// representation of a genome. Each connection includes its input node, output
// node, connection weight, and an indication of whether this connection is
// disabled
type ConnGene struct {
	From     int     `json:"from"`     // input node
	To       int     `json:"to"`       // output node
	Weight   float64 `json:"weight"`   // connection weight
	Disabled bool    `json:"disabled"` // true if disabled

	Generation int `json:"generation"` // generation in which this connection gene was created
	Innovation int `json:"innovation"` // innovation number of this gene
}

// NewConnGene returns a new instance of ConnGene, given the input and output
// node genes. By default, the connection is enabled.
func NewConnGene(from, to int, weight float64) *ConnGene {
	return &ConnGene{from, to, weight, false, 0, 0}
}

// Copy returns a deep copy of this connection gene.
func (c *ConnGene) Copy() *ConnGene {
	return &ConnGene{
		From:     c.From,
		To:       c.To,
		Weight:   c.Weight,
		Disabled: c.Disabled,
	}
}

// String returns the string representation of this connection.
func (c *ConnGene) String() string {
	connectivity := fmt.Sprintf("{%.3f}", c.Weight)
	if c.Disabled {
		connectivity = " / "
	}
	return fmt.Sprintf("[%d]-%s->[%d]", c.From, connectivity, c.To)
}

// Genome encodes the weights and topology of the output network as a collection
// of nodes and connection genes.
type Genome struct {
	ID          int         `json:"id"`          // genome ID
	SpeciesID   int         `json:"speciesID"`   // genome's species ID
	NodeGenes   []*NodeGene `json:"nodeGenes"`   // all nodes
	InputNodes  []*NodeGene `json:"inputNodes"`  // input nodes
	HiddenNodes []*NodeGene `json:"hiddenNodes"` // hidden nodes
	OutputNodes []*NodeGene `json:"outputNodes"` // output nodes
	ConnGenes   []*ConnGene `json:"connGenes"`   // connections in the genome
	Fitness     float64     `json:"fitness"`     // fitness score

	evaluated bool // true if already evaluated
}

// NewFCGenome returns an instance of initial Genome with fully connected input
// and output layers.
func NewFCGenome(id, numInputs, numOutputs int, initFitness float64, outputActivation string) *Genome {
	nodeGenes := make([]*NodeGene, 0, numInputs+numOutputs)
	inputNodes := make([]*NodeGene, 0, numInputs)
	outputNodes := make([]*NodeGene, 0, numOutputs)
	connGenes := make([]*ConnGene, 0, numInputs*numOutputs)
	innov := Innovation

	for i := 0; i < numInputs; i++ {
		inputNode := NewNodeGene(i, "input", ActivationSet["identity"])
		nodeGenes = append(nodeGenes, inputNode)
		inputNodes = append(inputNodes, inputNode)
	}
	for i := numInputs; i < numInputs+numOutputs; i++ {
		outputNode := NewNodeGene(i, "output", ActivationSet[outputActivation])
		for j := 0; j < numInputs; j++ {
			c := NewConnGene(j, i, rand.NormFloat64()*NeatConfig.InitConnWeight)
			c.Innovation = innov
			innov++
			connGenes = append(connGenes, c)
		}
		nodeGenes = append(nodeGenes, outputNode)
		outputNodes = append(outputNodes, outputNode)
	}
	return &Genome{
		ID:          id,
		SpeciesID:   -1,
		NodeGenes:   nodeGenes,
		InputNodes:  inputNodes,
		HiddenNodes: make([]*NodeGene, 0),
		OutputNodes: outputNodes,
		ConnGenes:   connGenes,
		Fitness:     initFitness,
		evaluated:   false,
	}
}

// NewGenome returns an instance of initial Genome with no initial connections.
func NewGenome(id, numInputs, numOutputs int, initFitness float64, outputActivation string) *Genome {
	nodeGenes := make([]*NodeGene, 0, numInputs+numOutputs)
	inputNodes := make([]*NodeGene, 0, numInputs)
	outputNodes := make([]*NodeGene, 0, numOutputs)

	for i := 0; i < numInputs; i++ {
		inputNode := NewNodeGene(i, "input", ActivationSet["identity"])
		nodeGenes = append(nodeGenes, inputNode)
		inputNodes = append(inputNodes, inputNode)
	}
	for i := numInputs; i < numInputs+numOutputs; i++ {
		outputNode := NewNodeGene(i, "output", ActivationSet[outputActivation])
		nodeGenes = append(nodeGenes, outputNode)
		outputNodes = append(outputNodes, outputNode)
	}
	return &Genome{
		ID:          id,
		SpeciesID:   -1,
		NodeGenes:   nodeGenes,
		InputNodes:  inputNodes,
		HiddenNodes: make([]*NodeGene, 0),
		OutputNodes: outputNodes,
		ConnGenes:   make([]*ConnGene, 0),
		Fitness:     initFitness,
		evaluated:   false,
	}
}

// Copy returns a deep copy of this genome.
func (g *Genome) Copy() *Genome {
	return &Genome{
		ID:        g.ID,
		SpeciesID: g.SpeciesID,
		NodeGenes: func() []*NodeGene {
			copies := make([]*NodeGene, len(g.NodeGenes))
			for i := range copies {
				copies[i] = g.NodeGenes[i].Copy()
			}
			return copies
		}(),
		ConnGenes: func() []*ConnGene {
			copies := make([]*ConnGene, len(g.ConnGenes))
			for i := range copies {
				copies[i] = g.ConnGenes[i].Copy()
			}
			return copies
		}(),
		Fitness:   g.Fitness,
		evaluated: g.evaluated,
	}
}

// String returns the string representation of the genome.
func (g *Genome) String() string {
	str := fmt.Sprintf("Genome(%d, %.3f):\n", g.ID, g.Fitness)
	str += "Nodes (\n"
	for _, node := range g.NodeGenes {
		str += "  " + node.String() + "\n"
	}
	str += ")\n"
	str += "Connections (\n"
	for _, conn := range g.ConnGenes {
		str += "  " + conn.String() + "\n"
	}
	str += ")"
	return str
}

// Evaluate takes an evaluation function and evaluates its fitness. Only perform
// the evaluation if it hasn't yet. If the lamarckian indicator is true, encode
// the phenotype neural network back into the genome.
func (g *Genome) Evaluate(evaluate EvaluationFunc) {
	if g.evaluated {
		return
	}
	nn := NewNeuralNetwork(g)
	g.Fitness = evaluate(nn)
	g.evaluated = true
}

// ExportJSON exports a JSON file that contains this genome's information. If
// the argument format indicator is true, the exported JSON file will be
// formatted with indentations.
func (g *Genome) ExportJSON(format bool) error {
	// create a new json file
	filename := fmt.Sprintf("genome_%d_%d.json", g.ID, time.Now().UnixNano())
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	if format {
		encoder.SetIndent("", "\t")
	}
	if err = encoder.Encode(g); err != nil {
		return err
	}

	return nil
}

func ImportJSON(jsonReader io.Reader) (*Genome, error) {
	g := &Genome{}
	decoder := json.NewDecoder(jsonReader)
	if err := decoder.Decode(g); err != nil {
		return &Genome{}, err
	}
	for _, node := range g.NodeGenes {
		node.Activation = ActivationSet[node.Activation.Name]
	}
	return g, nil
}

func (g *Genome) MutateActFunc(id int, acts []*ActivationFunc) {
	rNode := g.HiddenNodes[rand.Intn(len(g.HiddenNodes))]
	rNode.Activation = acts[rand.Intn(len(acts))]
	//adjust the connections to the new id that changed, when setting different activation function
	for _, conn := range g.ConnGenes {
		if conn.From == rNode.ID {
			conn.From = id
			conn.Innovation = Innovation
			Innovation++
		}
		if conn.To == rNode.ID {
			conn.To = id
			conn.Innovation = Innovation
			Innovation++
		}
	}
	rNode.ID = id
	g.evaluated = false
}

// Disable/ReEnable Connections
func (g *Genome) MutateDisEnConn(enRate, disRate float64) {
	for _, conn := range g.ConnGenes {
		if conn.Disabled {
			if rand.Float64() < enRate {
				conn.Disabled = false
				g.evaluated = false
			}
		} else {
			if rand.Float64() < disRate {
				conn.Disabled = true
				g.evaluated = false
			}
		}
	}
}

// MutatePerturb mutates the genome by perturbation of its weights by the
// argument rate by the given mutation range (called rang, because range is a keyword).
func (g *Genome) MutatePerturb(rate, rang, capt float64) {
	// perturb connection weights
	for _, conn := range g.ConnGenes {
		if rand.Float64() < rate {
			g.evaluated = false
			conn.Weight += rand.NormFloat64() * rang
			if conn.Weight > capt {
				conn.Weight = capt
			} else if conn.Weight < (-1.0 * capt) {
				conn.Weight = (-1.0 * capt)
			}
		}
	}
}

// MutateAddNode mutates the genome by adding a node with the argument
// activation function.
func (g *Genome) MutateAddNode(id int, activation *ActivationFunc) {
	// add node between two connected nodes, by randomly selecting a connection;
	// only applied if there are connections in the genome
	g.evaluated = false

	//get only active connections
	active := make([]*ConnGene, 0)
	for _, conn := range g.ConnGenes {
		if !conn.Disabled {
			active = append(active, conn)
		}
	}
	if len(active) < 1 {
		// no enabled connections so there is no point in adding a node now
		return
	}
	selected := active[rand.Intn(len(active))]
	newNode := NewNodeGene(len(g.NodeGenes), "hidden", activation)

	g.NodeGenes = append(g.NodeGenes, newNode)
	g.HiddenNodes = append(g.HiddenNodes, newNode)
	g.ConnGenes = append(g.ConnGenes,
		&ConnGene{selected.From, newNode.ID, 1.0, false, 0, Innovation},
		&ConnGene{newNode.ID, selected.To, selected.Weight, false, 0, Innovation})
	Innovation += 2
	selected.Disabled = true
}

// MutateAddConn mutates the genome by adding a connection.
func (g *Genome) MutateAddConn() {
	// add connection between two disconnected nodes; only applied if the selected
	// nodes are not connected yet
	tries := 5
	g.evaluated = false

	//TODO: improve determining the selected nodes by subtracting forbidden node types from the others
	for try := 0; try < tries; try++ {
		selectedNode0 := g.NodeGenes[rand.Intn(len(g.NodeGenes))]
		selectedNode1 := g.NodeGenes[rand.Intn(len(g.NodeGenes))]

		if selectedNode1.Type == "input" ||
			selectedNode0.Type == "output" {
			continue
		}

		for _, conn := range g.ConnGenes {
			if conn.From == selectedNode0.ID && conn.To == selectedNode1.ID {
				continue
			}
		}

		g.ConnGenes = append(g.ConnGenes, &ConnGene{selectedNode0.ID,
			selectedNode1.ID, rand.NormFloat64() * NeatConfig.InitConnWeight, false, 0, Innovation})
		Innovation++
		return
	}

}

// Crossover returns a new child genome by performing crossover between the two
// argument genomes.
//
// innovations is a temporary dictionary for the child genome's connection
// genes; it essentially stores all connection genes that will be contained
// in the child genome.
//
// Initially, all of one parent genome's connections are recorded to
// innovations. Then, as the other parent genome's connections are added, it
// checks if each connection already exists; if it does, swap with the other
// parent's connection by 50% chance. Otherwise, append the new connection.
func Crossover(id int, g0, g1 *Genome, initFitness float64) *Genome {
	innovations := make(map[[2]int]*ConnGene)
	child := g0.Copy()
	child.Fitness = initFitness
	child.ID = id
	for _, conn := range g0.ConnGenes {
		innovations[[2]int{conn.From, conn.To}] = conn
	}
	for _, conn := range g1.ConnGenes {
		innov := [2]int{conn.From, conn.To}
		if innovations[innov] != nil {
			if rand.Float64() < 0.5 {
				innovations[innov] = conn
			}
		}
	}
	// copy connection genes
	connGenes := make([]*ConnGene, 0, len(innovations))
	for _, conn := range innovations {
		connGenes = append(connGenes, conn.Copy())
	}
	child.ConnGenes = connGenes

	return child
}

// Compatibility computes the compatibility distance between two argument
// genomes.
//
// Compatibility distance of two genomes is utilized for differentiating their
// species during speciation. The distance is computed as follows:
//
//	d = c0 * U + c1 * W
//
// c0, c1, are hyperparameter coefficients, and U, W are respectively number of
// unmatching genes, and the average weight differences of matching genes. This
// approach is a slightly modified version of Dr. Kenneth Stanley's original
// approach in which unmatching genes are separated into excess and disjoint
// genes.
func Compatibility(g0, g1 *Genome, c0, c1 float64) float64 {
	innov0 := make(map[[2]int]*ConnGene) // innovations in g0
	innov1 := make(map[[2]int]*ConnGene) // innovations in g1

	for _, conn := range g0.ConnGenes {
		innov0[[2]int{conn.From, conn.To}] = conn
	}

	for _, conn := range g1.ConnGenes {
		innov1[[2]int{conn.From, conn.To}] = conn
	}

	matching := make(map[*ConnGene]*ConnGene) // pairs of matching genes
	unmatchingCount := 0                      // unmatching gene counter

	// look for matching/unmatching genes from g1's innovations; if a connection
	// in g0 is not one of g1's innovations, increment unmatching counter.
	// Otherwise, add the connection to matching
	for _, conn := range g0.ConnGenes {
		innov := innov1[[2]int{conn.From, conn.To}]
		if innov == nil {
			unmatchingCount++
		} else {
			matching[innov] = conn
		}
	}

	// repeat for g0's innovations, to count unmatching connection genes for g1.
	for _, conn := range g1.ConnGenes {
		innov := innov0[[2]int{conn.From, conn.To}]
		if innov == nil {
			unmatchingCount++
		}
	}

	// compute average weight differences of matching genes
	diffSum := 0.0
	matchingCount := len(matching)
	for conn0, conn1 := range matching {
		diffSum += math.Abs(conn0.Weight - conn1.Weight)
	}
	avgDiff := diffSum / float64(matchingCount)
	if matchingCount == 0 {
		avgDiff = 0.0
	}
	return c0*float64(unmatchingCount) + c1*avgDiff
}

// ComparisonFunc is a type of function that returns a boolean value that
// indicates whether the first argument genome is better than the second one
// in terms of its fitness.
type ComparisonFunc func(g0, g1 *Genome) bool

// NewComparisonFunc returns a new comparison function
// We stick to maximizing the fitness
// if you wish to minimize it, you can still return 1/ff in your evaluation function
func NewComparisonFunc() ComparisonFunc {
	return func(g0, g1 *Genome) bool {
		return g0.Fitness > g1.Fitness
	}

}
