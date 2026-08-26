from typing import List, Dict, Any, Callable

def agent_loop(
    prompt: str,
    llm: Callable[[str], str],
    executor: Callable[[str], Any],
    max_retries: int = 3
) -> Dict[str, Any]:
    """
    Agent loop to generate SQL, execute it, and retry on failure.
    Records history of attempts.
    """
    current_prompt = prompt
    history = []
    
    for attempt in range(max_retries):
        sql = llm(current_prompt)
        if not sql:
            history.append({"attempt": attempt, "sql": sql, "error": "LLM returned empty SQL"})
            break
            
        try:
            result = executor(sql)
            if not result:
                raise ValueError("Execution returned empty result")
                
            history.append({"attempt": attempt, "sql": sql, "error": None})
            return {"success": True, "result": result, "history": history, "final_sql": sql}
            
        except Exception as e:
            error_msg = str(e)
            history.append({"attempt": attempt, "sql": sql, "error": error_msg})
            current_prompt += f"\nPrevious attempt failed with error: {error_msg}. Please fix the SQL."
            
    return {"success": False, "result": None, "history": history, "final_sql": None}
