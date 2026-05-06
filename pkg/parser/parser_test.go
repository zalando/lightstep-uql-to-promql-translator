package parser

import (
	"testing"
)

func TestSampleQuery(t *testing.T) {
	query := `spans count
	| delta
	| filter (operation="my_operation" && service == $my_variable) && contains(my_val, "some_text") || defined(my_defined_attr)
	| reduce 1m, 1m, max
	| group_by [my_attribute], sum
	| top 10, min, 30m`

	astQuery, err := Parse(query)
	if err != nil {
		t.Fatalf("parser error %s", err.Error())
	}

	output := astQuery.ToXml()

	expected := `<Query>
  <Type>default</Type>
  <Pipeline>
    <FetchStageSpans>
      <FetchType>count</FetchType>
    </FetchStageSpans>
    <AlignerStageDelta></AlignerStageDelta>
    <ModifierStageFilter>
      <Expr>
        <InfixExpression>
          <Operation>||</Operation>
          <LeftExpr>
            <InfixExpression>
              <Operation>&amp;&amp;</Operation>
              <LeftExpr>
                <InfixExpression>
                  <Operation>&amp;&amp;</Operation>
                  <LeftExpr>
                    <InfixExpression>
                      <Operation>==</Operation>
                      <LeftExpr>
                        <Identifier>
                          <Value>operation</Value>
                        </Identifier>
                      </LeftExpr>
                      <RightExpr>
                        <StringLiteral>
                          <Value>my_operation</Value>
                        </StringLiteral>
                      </RightExpr>
                    </InfixExpression>
                  </LeftExpr>
                  <RightExpr>
                    <InfixExpression>
                      <Operation>==</Operation>
                      <LeftExpr>
                        <Identifier>
                          <Value>service</Value>
                        </Identifier>
                      </LeftExpr>
                      <RightExpr>
                        <TemplateVariable>
                          <Value>$my_variable</Value>
                        </TemplateVariable>
                      </RightExpr>
                    </InfixExpression>
                  </RightExpr>
                </InfixExpression>
              </LeftExpr>
              <RightExpr>
                <InfixExpression>
                  <Operation>contains</Operation>
                  <LeftExpr>
                    <Identifier>
                      <Value>my_val</Value>
                    </Identifier>
                  </LeftExpr>
                  <RightExpr>
                    <StringLiteral>
                      <Value>some_text</Value>
                    </StringLiteral>
                  </RightExpr>
                </InfixExpression>
              </RightExpr>
            </InfixExpression>
          </LeftExpr>
          <RightExpr>
            <PrefixExpression>
              <Operation>defined</Operation>
              <Expr>
                <Identifier>
                  <Value>my_defined_attr</Value>
                </Identifier>
              </Expr>
            </PrefixExpression>
          </RightExpr>
        </InfixExpression>
      </Expr>
    </ModifierStageFilter>
    <AlignerStageReduce>
      <InputWindow>
        <DurationLiteral>
          <Value>1m</Value>
        </DurationLiteral>
      </InputWindow>
      <OutputPeriod>
        <DurationLiteral>
          <Value>1m</Value>
        </DurationLiteral>
      </OutputPeriod>
      <Reducer>max</Reducer>
    </AlignerStageReduce>
    <ModifierStageGroupBy>
      <Labels>
        <Identifier>
          <Value>my_attribute</Value>
        </Identifier>
      </Labels>
      <Reducer>sum</Reducer>
    </ModifierStageGroupBy>
    <ModifierStageTop>
      <Labels></Labels>
      <Amount>
        <IntegerLiteral>
          <Value>10</Value>
        </IntegerLiteral>
      </Amount>
      <Reducer>min</Reducer>
      <Window>
        <DurationLiteral>
          <Value>30m</Value>
        </DurationLiteral>
      </Window>
    </ModifierStageTop>
  </Pipeline>
</Query>`

	if output != expected {
		t.Errorf("wrong output\n%s", output)
	}
}

func TestUnnamedJoin(t *testing.T) {
	query := `(
  		metric cpu.usage | delta | group_by [], sum;
    	metric cpu.requests | latest | group_by [], sum;
     ) | join left / right * 100, 0, 1`

	astQuery, err := Parse(query)
	if err != nil {
		t.Fatal(err)
	}

	output := astQuery.ToXml()

	expected := `<Query>
  <Type>unnamed_join</Type>
  <Pipeline></Pipeline>
  <UnnamedJoin>
    <Left>
      <Query>
        <Type>default</Type>
        <Pipeline>
          <FetchStageMetric>
            <MetricName>cpu.usage</MetricName>
          </FetchStageMetric>
          <AlignerStageDelta></AlignerStageDelta>
          <ModifierStageGroupBy>
            <Labels></Labels>
            <Reducer>sum</Reducer>
          </ModifierStageGroupBy>
        </Pipeline>
      </Query>
    </Left>
    <Right>
      <Query>
        <Type>default</Type>
        <Pipeline>
          <FetchStageMetric>
            <MetricName>cpu.requests</MetricName>
          </FetchStageMetric>
          <AlignerStageLatest></AlignerStageLatest>
          <ModifierStageGroupBy>
            <Labels></Labels>
            <Reducer>sum</Reducer>
          </ModifierStageGroupBy>
        </Pipeline>
      </Query>
    </Right>
    <Stages>
      <ModifierStageJoin>
        <Expr>
          <InfixExpression>
            <Operation>*</Operation>
            <LeftExpr>
              <InfixExpression>
                <Operation>/</Operation>
                <LeftExpr>
                  <Identifier>
                    <Value>left</Value>
                  </Identifier>
                </LeftExpr>
                <RightExpr>
                  <Identifier>
                    <Value>right</Value>
                  </Identifier>
                </RightExpr>
              </InfixExpression>
            </LeftExpr>
            <RightExpr>
              <IntegerLiteral>
                <Value>100</Value>
              </IntegerLiteral>
            </RightExpr>
          </InfixExpression>
        </Expr>
        <LeftDefault>
          <IntegerLiteral>
            <Value>0</Value>
          </IntegerLiteral>
        </LeftDefault>
        <RightDefault>
          <IntegerLiteral>
            <Value>1</Value>
          </IntegerLiteral>
        </RightDefault>
      </ModifierStageJoin>
    </Stages>
  </UnnamedJoin>
</Query>`

	if output != expected {
		t.Errorf("wrong output\n%s", output)
	}
}

func TestNamedJoin(t *testing.T) {
	query := `
	with
  		requests = metric cpu.requests | latest | group_by [], sum;
    	usage = metric cpu.usage | delta | group_by [], sum;
    join usage / requests * 100, usage=0
    | group_by [], sum`

	astQuery, err := Parse(query)
	if err != nil {
		t.Fatal(err)
	}

	output := astQuery.ToXml()

	expected := `<Query>
  <Type>named_join</Type>
  <Pipeline></Pipeline>
  <NamedJoin>
    <Queries>
      <NamedJoinPipeline>
        <Name>requests</Name>
        <Query>
          <Type>default</Type>
          <Pipeline>
            <FetchStageMetric>
              <MetricName>cpu.requests</MetricName>
            </FetchStageMetric>
            <AlignerStageLatest></AlignerStageLatest>
            <ModifierStageGroupBy>
              <Labels></Labels>
              <Reducer>sum</Reducer>
            </ModifierStageGroupBy>
          </Pipeline>
        </Query>
      </NamedJoinPipeline>
      <NamedJoinPipeline>
        <Name>usage</Name>
        <Query>
          <Type>default</Type>
          <Pipeline>
            <FetchStageMetric>
              <MetricName>cpu.usage</MetricName>
            </FetchStageMetric>
            <AlignerStageDelta></AlignerStageDelta>
            <ModifierStageGroupBy>
              <Labels></Labels>
              <Reducer>sum</Reducer>
            </ModifierStageGroupBy>
          </Pipeline>
        </Query>
        <Default>
          <IntegerLiteral>
            <Value>0</Value>
          </IntegerLiteral>
        </Default>
      </NamedJoinPipeline>
    </Queries>
    <JoinExpr>
      <InfixExpression>
        <Operation>*</Operation>
        <LeftExpr>
          <InfixExpression>
            <Operation>/</Operation>
            <LeftExpr>
              <Identifier>
                <Value>usage</Value>
              </Identifier>
            </LeftExpr>
            <RightExpr>
              <Identifier>
                <Value>requests</Value>
              </Identifier>
            </RightExpr>
          </InfixExpression>
        </LeftExpr>
        <RightExpr>
          <IntegerLiteral>
            <Value>100</Value>
          </IntegerLiteral>
        </RightExpr>
      </InfixExpression>
    </JoinExpr>
    <Stages>
      <ModifierStageGroupBy>
        <Labels></Labels>
        <Reducer>sum</Reducer>
      </ModifierStageGroupBy>
    </Stages>
  </NamedJoin>
</Query>`

	if output != expected {
		t.Errorf("wrong output\n%s", output)
	}
}

func TestMultiplePointExpressions(t *testing.T) {
	query := `spans latency
| delta 10m
| filter operation == "test op"
| group_by [], sum
| point percentile(value, 50.0), percentile(value, 75.0), percentile(value, 95.0)`

	astQuery, err := Parse(query)
	if err != nil {
		t.Fatalf("parser error %s", err.Error())
	}

	output := astQuery.ToXml()

	expected := `<Query>
  <Type>default</Type>
  <Pipeline>
    <FetchStageSpans>
      <FetchType>latency</FetchType>
    </FetchStageSpans>
    <AlignerStageDelta>
      <InputWindow>
        <DurationLiteral>
          <Value>10m</Value>
        </DurationLiteral>
      </InputWindow>
    </AlignerStageDelta>
    <ModifierStageFilter>
      <Expr>
        <InfixExpression>
          <Operation>==</Operation>
          <LeftExpr>
            <Identifier>
              <Value>operation</Value>
            </Identifier>
          </LeftExpr>
          <RightExpr>
            <StringLiteral>
              <Value>test op</Value>
            </StringLiteral>
          </RightExpr>
        </InfixExpression>
      </Expr>
    </ModifierStageFilter>
    <ModifierStageGroupBy>
      <Labels></Labels>
      <Reducer>sum</Reducer>
    </ModifierStageGroupBy>
    <ModifierStagePoint>
      <Expressions>
        <InfixExpression>
          <Operation>percentile</Operation>
          <LeftExpr>
            <Identifier>
              <Value>value</Value>
            </Identifier>
          </LeftExpr>
          <RightExpr>
            <FloatLiteral>
              <Value>50.0</Value>
            </FloatLiteral>
          </RightExpr>
        </InfixExpression>
        <InfixExpression>
          <Operation>percentile</Operation>
          <LeftExpr>
            <Identifier>
              <Value>value</Value>
            </Identifier>
          </LeftExpr>
          <RightExpr>
            <FloatLiteral>
              <Value>75.0</Value>
            </FloatLiteral>
          </RightExpr>
        </InfixExpression>
        <InfixExpression>
          <Operation>percentile</Operation>
          <LeftExpr>
            <Identifier>
              <Value>value</Value>
            </Identifier>
          </LeftExpr>
          <RightExpr>
            <FloatLiteral>
              <Value>95.0</Value>
            </FloatLiteral>
          </RightExpr>
        </InfixExpression>
      </Expressions>
    </ModifierStagePoint>
  </Pipeline>
</Query>`

	if output != expected {
		t.Errorf("wrong output\n%s", output)
	}
}
